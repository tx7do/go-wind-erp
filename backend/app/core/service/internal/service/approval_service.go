package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// approvalViewerUserID 从 viewer context 提取调用者用户 ID（与 data 层
// viewerUserIDFromContext 同语义；service 层直接读 viewer 包）。
func approvalViewerUserID(ctx context.Context) (uint32, bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist || vc == nil {
		return 0, false
	}
	uid := vc.UserID()
	if uid == 0 {
		return 0, false
	}
	return uint32(uid), true
}

// ApprovalRequestService 审批请求服务
type ApprovalRequestService struct {
	approvalV1.UnimplementedApprovalRequestServiceServer

	log *log.Helper

	approvalRequestRepo *data.ApprovalRequestRepo

	// 采购联动：biz_type=purchase_order 的审批通过/驳回时回写采购单状态。
	// 经 PurchaseOrderService 动作走单一通路（自审守卫、应付生成等
	// 服务层逻辑全部生效），而非直连 repo 绕过。
	purchaseOrderService *PurchaseOrderService
}

func NewApprovalRequestService(
	ctx *bootstrap.Context,
	approvalRequestRepo *data.ApprovalRequestRepo,
	purchaseOrderService *PurchaseOrderService,
) *ApprovalRequestService {
	svc := &ApprovalRequestService{
		log:                  ctx.NewLoggerHelper("approval_request/service/core-service"),
		approvalRequestRepo:  approvalRequestRepo,
		purchaseOrderService: purchaseOrderService,
	}

	return svc
}

func (s *ApprovalRequestService) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalRequestResponse, error) {
	return s.approvalRequestRepo.List(ctx, req)
}

func (s *ApprovalRequestService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.CountApprovalRequestResponse, error) {
	count, err := s.approvalRequestRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &approvalV1.CountApprovalRequestResponse{Count: uint64(count)}, nil
}

func (s *ApprovalRequestService) Get(ctx context.Context, req *approvalV1.GetApprovalRequestRequest) (*approvalV1.ApprovalRequest, error) {
	return s.approvalRequestRepo.Get(ctx, req)
}

func (s *ApprovalRequestService) Create(ctx context.Context, req *approvalV1.CreateApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}

	// 标题与业务类型必须由服务端校验，避免空值进入审批流
	if err := validateApprovalFields(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.approvalRequestRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// validateApprovalFields 校验审批必填字段。
func validateApprovalFields(a *approvalV1.ApprovalRequest) error {
	if a == nil {
		return approvalV1.ErrorBadRequest("invalid parameter")
	}
	if a.GetTitle() == "" {
		return approvalV1.ErrorBadRequest("title is required")
	}
	if a.GetBizType() == "" {
		return approvalV1.ErrorBadRequest("biz_type is required")
	}
	return nil
}

func (s *ApprovalRequestService) Delete(ctx context.Context, req *approvalV1.DeleteApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}

	// 仅已撤销的请求可删除：终态审批记录是审计凭据，不可事后抹除。
	old, err := s.approvalRequestRepo.Get(ctx, &approvalV1.GetApprovalRequestRequest{
		QueryBy: &approvalV1.GetApprovalRequestRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	if old.GetStatus() != approvalV1.ApprovalRequest_CANCELLED {
		return nil, approvalV1.ErrorConflict("only cancelled approval requests can be deleted")
	}

	if err := s.approvalRequestRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// transition 执行带状态机校验的状态迁移：先读当前状态校验 from→to 合法，
// 再由 repo 做条件更新（仅 from 状态可迁移）保证并发安全。
func (s *ApprovalRequestService) transition(
	ctx context.Context,
	id uint32,
	to approvalV1.ApprovalRequest_Status,
	comment *string,
) (*emptypb.Empty, error) {
	old, err := s.approvalRequestRepo.Get(ctx, &approvalV1.GetApprovalRequestRequest{
		QueryBy: &approvalV1.GetApprovalRequestRequest_Id{Id: id},
	})
	if err != nil {
		return nil, err
	}

	if !validateApprovalTransition(old.GetStatus(), to) {
		return nil, approvalV1.ErrorConflict("approval request is not pending")
	}

	// 职责分离：申请人不得审批自己的请求（撤销走 Cancel，不受此限）。
	if to == approvalV1.ApprovalRequest_APPROVED || to == approvalV1.ApprovalRequest_REJECTED {
		if caller, ok := approvalViewerUserID(ctx); ok && caller == old.GetApplicantId() {
			return nil, approvalV1.ErrorConflict("applicant cannot approve or reject their own request")
		}
	}

	if _, err := s.approvalRequestRepo.TransitionStatus(ctx, id, old.GetStatus(), to, comment); err != nil {
		return nil, err
	}

	// 采购联动：审批结果回写采购单。回写失败不回滚审批（审批已是事实），
	// 仅记录，采购单可经管理端动作对齐。
	s.syncPurchaseOrder(ctx, old, to)

	return &emptypb.Empty{}, nil
}

// syncPurchaseOrder 对 biz_type=purchase_order 的审批，将其结果同步到
// 采购单（SUBMITTED→APPROVED/REJECTED）。biz_ref 形如 "purchase_order:{id}"。
func (s *ApprovalRequestService) syncPurchaseOrder(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) {
	if old.GetBizType() != "purchase_order" {
		return
	}
	raw := strings.TrimPrefix(old.GetBizRef(), "purchase_order:")
	poID, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		s.log.Errorf("parse purchase order ref failed: %s", old.GetBizRef())
		return
	}

	// 经 PO 服务动作同步：享受与直审完全相同的守卫（自审拦截、
	// 原子迁移、获批生成应付单）。
	if to == approvalV1.ApprovalRequest_APPROVED {
		if _, err := s.purchaseOrderService.Approve(ctx, &procurementV1.ApprovePurchaseOrderRequest{
			Id: uint32(poID),
		}); err != nil {
			s.log.Errorf("sync purchase order %d to APPROVED failed: %s", poID, err.Error())
		}
		return
	}
	if to == approvalV1.ApprovalRequest_REJECTED {
		if _, err := s.purchaseOrderService.Reject(ctx, &procurementV1.RejectPurchaseOrderRequest{
			Id: uint32(poID),
		}); err != nil {
			s.log.Errorf("sync purchase order %d to REJECTED failed: %s", poID, err.Error())
		}
	}
}

// Approve 通过审批（仅 PENDING）。
func (s *ApprovalRequestService) Approve(ctx context.Context, req *approvalV1.ApproveApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), approvalV1.ApprovalRequest_APPROVED, req.Comment)
}

// Reject 驳回审批（仅 PENDING）。
func (s *ApprovalRequestService) Reject(ctx context.Context, req *approvalV1.RejectApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), approvalV1.ApprovalRequest_REJECTED, req.Comment)
}

// Cancel 撤销审批（仅 PENDING 且仅申请人本人）。
func (s *ApprovalRequestService) Cancel(ctx context.Context, req *approvalV1.CancelApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}

	old, err := s.approvalRequestRepo.Get(ctx, &approvalV1.GetApprovalRequestRequest{
		QueryBy: &approvalV1.GetApprovalRequestRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	if !validateApprovalTransition(old.GetStatus(), approvalV1.ApprovalRequest_CANCELLED) {
		return nil, approvalV1.ErrorConflict("approval request is not pending")
	}

	// 仅申请人本人可撤销自己的请求
	callerUserID, hasUser := approvalViewerUserID(ctx)
	if !hasUser || callerUserID != old.GetApplicantId() {
		return nil, approvalV1.ErrorConflict("only the applicant can cancel this request")
	}

	if err := s.approvalRequestRepo.CancelAsApplicant(ctx, req.GetId(), callerUserID); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
