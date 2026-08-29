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
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
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

	// 多级审批：流程模板仓储（级进角色校验 + 创建时快照）
	approvalFlowRepo *data.ApprovalFlowRepo

	// 采购联动：biz_type=PURCHASE_ORDER 的审批通过/驳回时回写采购单状态。
	// 经 PurchaseOrderService 动作走单一通路（自审守卫、应付生成等
	// 服务层逻辑全部生效），而非直连 repo 绕过。
	purchaseOrderService *PurchaseOrderService

	// 销售联动：biz_type=SALES_ORDER 的审批通过/驳回时回写销售单状态。
	// 经 SalesOrderService 动作走单一通路（自审守卫、应收生成等
	// 服务层逻辑全部生效），而非直连 repo 绕过。
	salesOrderService *SalesOrderService

	// 付款联动：biz_type=PAYMENT 的审批通过/拒绝时驱动付款入账。
	paymentService *PaymentService

	// 收款联动：biz_type=RECEIPT 的审批通过/拒绝时驱动收款入账。
	receiptService *ReceiptService

	// 审结站内信通知（申请人）。
	notifier *approvalNotifier
}

func NewApprovalRequestService(
	ctx *bootstrap.Context,
	approvalRequestRepo *data.ApprovalRequestRepo,
	approvalFlowRepo *data.ApprovalFlowRepo,
	purchaseOrderService *PurchaseOrderService,
	salesOrderService *SalesOrderService,
	paymentService *PaymentService,
	receiptService *ReceiptService,
	messageRepo *data.InternalMessageRepo,
	recipientRepo *data.InternalMessageRecipientRepo,
) *ApprovalRequestService {
	l := ctx.NewLoggerHelper("approval_request/service/core-service")
	svc := &ApprovalRequestService{
		log:                  l,
		approvalRequestRepo:  approvalRequestRepo,
		approvalFlowRepo:     approvalFlowRepo,
		purchaseOrderService: purchaseOrderService,
		salesOrderService:    salesOrderService,
		paymentService:       paymentService,
		receiptService:       receiptService,
		notifier: &approvalNotifier{
			messageRepo:   messageRepo,
			recipientRepo: recipientRepo,
			log:           l,
		},
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

	// 多级审批快照在仓储层单点完成（PO/SO/付款/收款/补货提交路径均
	// 直连仓储建单），此处无需重复。
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

	// 多级审批：非终级通过 = 级进（保持 PENDING，通知下一级），不触发
	// 业务回写；终级通过或驳回才走终态迁移 + 业务联动。
	if to == approvalV1.ApprovalRequest_APPROVED && old.GetTotalSteps() > 1 {
		if err := s.advanceOrFinalize(ctx, old, comment); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	// 业务联动先行：审批结果按 biz_type 分发回写（销售/采购/付款/收款/
	// 补货）。业务守卫（自审/信用额度/状态机）拒绝 → 整个审批动作失败，
	// 审批单保持 PENDING 可重试——审批通过必须是业务生效的同义词。
	if serr := s.syncBusiness(ctx, old, to); serr != nil {
		return nil, serr
	}

	if _, err := s.approvalRequestRepo.TransitionStatus(ctx, id, old.GetStatus(), to, comment); err != nil {
		return nil, err
	}

	// 审结通知申请人（尽力而为）。
	if nerr := s.notifier.notifyResolved(ctx, old, to == approvalV1.ApprovalRequest_APPROVED); nerr != nil {
		s.log.Errorf("notify approval resolved failed: %s", nerr.Error())
	}

	return &emptypb.Empty{}, nil
}

// advanceOrFinalize 多级审批的通过动作：校验当前用户持有本级审批角色，
// 非终级 → current_step+1 保持 PENDING 并通知下一级候选审批人；
// 终级 → 落 APPROVED 终态（业务联动与审结通知由调用方 transition 继续）。
func (s *ApprovalRequestService) advanceOrFinalize(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	comment *string,
) error {
	current := old.GetCurrentStep()
	total := old.GetTotalSteps()

	// 级角色校验：持有本级 role_code 的用户方可审批（申请人自审已在
	// transition 前置拦截）。流程或级定义缺失（被删改）→ 放行为终态，
	// 避免在途单卡死。
	if old.GetFlowId() != 0 {
		roleCode, err := s.approvalFlowRepo.GetStepRole(ctx, old.GetFlowId(), current)
		if err != nil {
			return err
		}
		if roleCode != "" {
			caller, ok := approvalViewerUserID(ctx)
			if !ok {
				return approvalV1.ErrorBadRequest("caller context required")
			}
			holds, err := s.approvalFlowRepo.UserHoldsRole(ctx, caller, roleCode)
			if err != nil {
				return err
			}
			if !holds {
				return approvalV1.ErrorForbidden(
					"第 %d 级需持有角色 %s 的用户审批", current, roleCode)
			}
		}
	}

	if current < total {
		// 级进：原子推进（并发安全），通知下一级候选审批人（尽力而为）。
		if err := s.approvalRequestRepo.AdvanceStep(ctx, old.GetId(), current, current+1, comment); err != nil {
			return err
		}
		if roleCode, err := s.approvalFlowRepo.GetStepRole(ctx, old.GetFlowId(), current+1); err == nil && roleCode != "" {
			if userIDs, uerr := s.approvalFlowRepo.UserIDsByRole(ctx, roleCode); uerr == nil {
				if nerr := s.notifier.notifyStepApprovers(ctx, old, current+1, total, userIDs); nerr != nil {
					s.log.Errorf("notify next step approvers failed: %s", nerr.Error())
				}
			}
		}
		return nil
	}

	// 终级通过：业务联动先行（守卫拒绝→错误上抛，级数不推进），成功再
	// 落 APPROVED 终态 + 审结通知。
	if serr := s.syncBusiness(ctx, old, approvalV1.ApprovalRequest_APPROVED); serr != nil {
		return serr
	}
	if _, err := s.approvalRequestRepo.TransitionStatus(ctx, old.GetId(), old.GetStatus(), approvalV1.ApprovalRequest_APPROVED, comment); err != nil {
		return err
	}

	if nerr := s.notifier.notifyResolved(ctx, old, true); nerr != nil {
		s.log.Errorf("notify approval resolved failed: %s", nerr.Error())
	}
	return nil
}

// syncBusiness 按 biz_type 分发审批结果联动。
func (s *ApprovalRequestService) syncBusiness(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	switch old.GetBizType() {
	case poApprovalBizType:
		return s.syncPurchaseOrder(ctx, old, to)
	case soApprovalBizType:
		return s.syncSalesOrder(ctx, old, to)
	case paymentApprovalBizType:
		return s.syncPayment(ctx, old, to)
	case receiptApprovalBizType:
		return s.syncReceipt(ctx, old, to)
	case replenishmentBizType:
		return s.syncReplenishment(ctx, old, to)
	}
	return nil
}

// syncReplenishment 对 biz_type=REPLENISHMENT 的审批，通过后自动创建
// 草稿采购单（供应商史缺失时不建单，仅记录——建议单仍留痕）并通知
// 申请人草稿已就绪。biz_ref 形如 "REPLENISHMENT:{warehouse}:{sku}"。
func (s *ApprovalRequestService) syncReplenishment(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	if to != approvalV1.ApprovalRequest_APPROVED {
		return nil
	}

	raw := strings.TrimPrefix(old.GetBizRef(), "REPLENISHMENT:")
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		s.log.Errorf("parse replenishment ref failed: %s", old.GetBizRef())
		return approvalV1.ErrorBadRequest("invalid biz_ref")
	}

	po, err := s.purchaseOrderService.CreateReplenishmentDraft(ctx, parts[0], parts[1])
	if err != nil {
		s.log.Errorf("create replenishment draft for %s failed: %s", old.GetBizRef(), err.Error())
		return err
	}

	// 下游事件通知：草稿采购单已创建，待完善提交（失败仅记录，不阻塞）。
	if nerr := s.notifier.notifyReplenishmentDraft(ctx, old, po.GetPoNumber()); nerr != nil {
		s.log.Errorf("notify replenishment draft for %s failed: %s", old.GetBizRef(), nerr.Error())
	}
	return nil
}

// syncPayment 对 biz_type=PAYMENT 的审批，驱动付款入账/拒绝。
// biz_ref 形如 "PAYMENT:{id}"。
func (s *ApprovalRequestService) syncPayment(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	raw := strings.TrimPrefix(old.GetBizRef(), "PAYMENT:")
	paymentID, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		s.log.Errorf("parse payment ref failed: %s", old.GetBizRef())
		return approvalV1.ErrorBadRequest("invalid biz_ref")
	}

	switch to {
	case approvalV1.ApprovalRequest_APPROVED:
		if err := s.paymentService.ApplyApproved(ctx, uint32(paymentID)); err != nil {
			s.log.Errorf("apply payment %d failed: %s", paymentID, err.Error())
			return err
		}
	case approvalV1.ApprovalRequest_REJECTED:
		if err := s.paymentService.RejectApplied(ctx, uint32(paymentID)); err != nil {
			s.log.Errorf("reject payment %d failed: %s", paymentID, err.Error())
			return err
		}
	}
	return nil
}

// syncReceipt 对 biz_type=RECEIPT 的审批，驱动收款入账/拒绝。
// biz_ref 形如 "RECEIPT:{id}"。镜像 syncPayment。
func (s *ApprovalRequestService) syncReceipt(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	raw := strings.TrimPrefix(old.GetBizRef(), "RECEIPT:")
	receiptID, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		s.log.Errorf("parse receipt ref failed: %s", old.GetBizRef())
		return approvalV1.ErrorBadRequest("invalid biz_ref")
	}

	switch to {
	case approvalV1.ApprovalRequest_APPROVED:
		if err := s.receiptService.ApplyApproved(ctx, uint32(receiptID)); err != nil {
			s.log.Errorf("apply receipt %d failed: %s", receiptID, err.Error())
			return err
		}
	case approvalV1.ApprovalRequest_REJECTED:
		if err := s.receiptService.RejectApplied(ctx, uint32(receiptID)); err != nil {
			s.log.Errorf("reject receipt %d failed: %s", receiptID, err.Error())
			return err
		}
	}
	return nil
}

// syncPurchaseOrder 对 biz_type=PURCHASE_ORDER 的审批，将其结果同步到
// 采购单（SUBMITTED→APPROVED/REJECTED）。biz_ref 形如 "PURCHASE_ORDER:{id}"。
func (s *ApprovalRequestService) syncPurchaseOrder(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	if old.GetBizType() != "PURCHASE_ORDER" {
		return nil
	}
	raw := strings.TrimPrefix(old.GetBizRef(), "PURCHASE_ORDER:")
	poID, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		s.log.Errorf("parse purchase order ref failed: %s", old.GetBizRef())
		return approvalV1.ErrorBadRequest("invalid biz_ref")
	}

	// 经 PO 服务动作同步：享受与直审完全相同的守卫（自审拦截、
	// 原子迁移、获批生成应付单）。
	if to == approvalV1.ApprovalRequest_APPROVED {
		if _, err := s.purchaseOrderService.Approve(ctx, &procurementV1.ApprovePurchaseOrderRequest{
			Id: uint32(poID),
		}); err != nil {
			s.log.Errorf("sync purchase order %d to APPROVED failed: %s", poID, err.Error())
			return err
		}
		return nil
	}
	if to == approvalV1.ApprovalRequest_REJECTED {
		if _, err := s.purchaseOrderService.Reject(ctx, &procurementV1.RejectPurchaseOrderRequest{
			Id: uint32(poID),
		}); err != nil {
			s.log.Errorf("sync purchase order %d to REJECTED failed: %s", poID, err.Error())
			return err
		}
	}
	return nil
}

// syncSalesOrder 对 biz_type=SALES_ORDER 的审批，将其结果同步到
// 销售单（SUBMITTED→APPROVED/REJECTED）。biz_ref 形如 "SALES_ORDER:{id}"。
func (s *ApprovalRequestService) syncSalesOrder(
	ctx context.Context,
	old *approvalV1.ApprovalRequest,
	to approvalV1.ApprovalRequest_Status,
) error {
	if old.GetBizType() != soApprovalBizType {
		return nil
	}
	raw := strings.TrimPrefix(old.GetBizRef(), "SALES_ORDER:")
	soID, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		s.log.Errorf("parse sales order ref failed: %s", old.GetBizRef())
		return approvalV1.ErrorBadRequest("invalid biz_ref")
	}

	// 经 SO 服务动作同步：享受与直审完全相同的守卫（自审拦截、
	// 原子迁移、信用额度、获批生成应收单）。
	if to == approvalV1.ApprovalRequest_APPROVED {
		if _, err := s.salesOrderService.Approve(ctx, &salesV1.ApproveSalesOrderRequest{
			Id: uint32(soID),
		}); err != nil {
			s.log.Errorf("sync sales order %d to APPROVED failed: %s", soID, err.Error())
			return err
		}
		return nil
	}
	if to == approvalV1.ApprovalRequest_REJECTED {
		if _, err := s.salesOrderService.Reject(ctx, &salesV1.RejectSalesOrderRequest{
			Id: uint32(soID),
		}); err != nil {
			s.log.Errorf("sync sales order %d to REJECTED failed: %s", soID, err.Error())
			return err
		}
	}
	return nil
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
