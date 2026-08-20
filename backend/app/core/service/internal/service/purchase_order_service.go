package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// 采购审批联动约定：审批单 biz_type 固定为 "purchase_order"，
// biz_ref 形如 "purchase_order:{poId}"。审批中心通过/驳回时按此回写采购单状态。
const (
	poApprovalBizType = "purchase_order"
	poApprovalBizRef  = "purchase_order:%d"
)

// PurchaseOrderService 采购单服务
type PurchaseOrderService struct {
	procurementV1.UnimplementedPurchaseOrderServiceServer

	log *log.Helper

	purchaseOrderRepo   *data.PurchaseOrderRepo
	approvalRequestRepo *data.ApprovalRequestRepo
	payableRepo         *data.PayableRepo
}

func NewPurchaseOrderService(
	ctx *bootstrap.Context,
	purchaseOrderRepo *data.PurchaseOrderRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	payableRepo *data.PayableRepo,
) *PurchaseOrderService {
	svc := &PurchaseOrderService{
		log:                 ctx.NewLoggerHelper("purchase_order/service/core-service"),
		purchaseOrderRepo:   purchaseOrderRepo,
		approvalRequestRepo: approvalRequestRepo,
		payableRepo:         payableRepo,
	}

	return svc
}

func (s *PurchaseOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListPurchaseOrderResponse, error) {
	return s.purchaseOrderRepo.List(ctx, req)
}

func (s *PurchaseOrderService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.CountPurchaseOrderResponse, error) {
	count, err := s.purchaseOrderRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &procurementV1.CountPurchaseOrderResponse{Count: uint64(count)}, nil
}

func (s *PurchaseOrderService) Get(ctx context.Context, req *procurementV1.GetPurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	return s.purchaseOrderRepo.Get(ctx, req)
}

func (s *PurchaseOrderService) Create(ctx context.Context, req *procurementV1.CreatePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	// 校验明细并预计算金额（乘法溢出守卫 + 累加守卫）。
	if err := computeOrderAmounts(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.purchaseOrderRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Update 仅 DRAFT 可改（状态守卫）；携带明细时重算金额并整体替换。
func (s *PurchaseOrderService) Update(ctx context.Context, req *procurementV1.UpdatePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	old, err := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
		QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	// DRAFT 可改；REJECTED 支持驳回后修改重提。
	if old.GetStatus() != procurementV1.PurchaseOrder_DRAFT &&
		old.GetStatus() != procurementV1.PurchaseOrder_REJECTED {
		return nil, procurementV1.ErrorConflict("only draft or rejected purchase orders can be updated")
	}

	if err := computeOrderAmounts(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.purchaseOrderRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Delete 仅 DRAFT/CANCELLED 可删（终态审计记录与在途单不可抹除）。
func (s *PurchaseOrderService) Delete(ctx context.Context, req *procurementV1.DeletePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}

	old, err := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
		QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	if old.GetStatus() != procurementV1.PurchaseOrder_DRAFT &&
		old.GetStatus() != procurementV1.PurchaseOrder_CANCELLED {
		return nil, procurementV1.ErrorConflict("only draft or cancelled purchase orders can be deleted")
	}

	if err := s.purchaseOrderRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Submit 提交：DRAFT→SUBMITTED（原子迁移），并自动生成关联审批请求
// （申请人由审批 repo 从 viewer 推导）。
func (s *PurchaseOrderService) Submit(ctx context.Context, req *procurementV1.SubmitPurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), procurementV1.PurchaseOrder_SUBMITTED, true)
}

// Approve 通过：SUBMITTED→APPROVED（管理端直审；审批中心路径经
// approval 同步触发同一迁移）。
func (s *PurchaseOrderService) Approve(ctx context.Context, req *procurementV1.ApprovePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), procurementV1.PurchaseOrder_APPROVED, false)
}

// Reject 驳回：SUBMITTED→REJECTED。
func (s *PurchaseOrderService) Reject(ctx context.Context, req *procurementV1.RejectPurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), procurementV1.PurchaseOrder_REJECTED, false)
}

// Complete 完结：APPROVED→COMPLETED（到货完结）。
func (s *PurchaseOrderService) Complete(ctx context.Context, req *procurementV1.CompletePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), procurementV1.PurchaseOrder_COMPLETED, false)
}

// Cancel 取消：非终态→CANCELLED，仅发起人本人。
func (s *PurchaseOrderService) Cancel(ctx context.Context, req *procurementV1.CancelPurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}

	old, err := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
		QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	if !validatePurchaseOrderTransition(old.GetStatus(), procurementV1.PurchaseOrder_CANCELLED) {
		return nil, procurementV1.ErrorConflict("purchase order status transition not allowed")
	}

	// 仅发起人（created_by）可取消自己的单据。
	caller, ok := approvalViewerUserID(ctx)
	if !ok || caller != old.GetCreatedBy() {
		return nil, procurementV1.ErrorConflict("only the creator can cancel this purchase order")
	}

	if err := s.purchaseOrderRepo.TransitionStatus(ctx, req.GetId(), old.GetStatus(), procurementV1.PurchaseOrder_CANCELLED); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// transition 通用原子迁移：读-验-条件更新（0 行→409）。
// createApproval 为 true 时（Submit 路径）在迁移成功后生成审批请求；
// 审批生成失败不影响提交结果，仅记录（审批单可人工补建）。
func (s *PurchaseOrderService) transition(
	ctx context.Context,
	id uint32,
	to procurementV1.PurchaseOrder_Status,
	createApproval bool,
) (*emptypb.Empty, error) {
	old, err := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
		QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: id},
	})
	if err != nil {
		return nil, err
	}

	if !validatePurchaseOrderTransition(old.GetStatus(), to) {
		return nil, procurementV1.ErrorConflict("purchase order status transition not allowed")
	}

	// 职责分离：管理端直审不允许发起人自审自（审批中心路径已由审批域守卫）。
	if to == procurementV1.PurchaseOrder_APPROVED || to == procurementV1.PurchaseOrder_REJECTED {
		if caller, ok := approvalViewerUserID(ctx); ok && caller == old.GetCreatedBy() {
			return nil, procurementV1.ErrorConflict("creator cannot approve or reject their own purchase order")
		}
	}

	if err := s.purchaseOrderRepo.TransitionStatus(ctx, id, old.GetStatus(), to); err != nil {
		return nil, err
	}

	// 财务联动：采购单获批即生成全额应付单（手工建账之外的自动来源）。
	// 生成失败不影响审批结果，仅记录（可手工补建）。
	if to == procurementV1.PurchaseOrder_APPROVED {
		if _, err := s.payableRepo.Create(ctx, &financeV1.CreatePayableRequest{
			Data: &financeV1.Payable{
				PoRef:        trans.Ptr(fmt.Sprintf(poApprovalBizRef, id)),
				SupplierCode: trans.Ptr(old.GetSupplierCode()),
				Amount:       trans.Ptr(old.GetTotalAmount()),
			},
		}); err != nil {
			s.log.Errorf("create payable for purchase order %d failed: %s", id, err.Error())
		}
	}

	if createApproval && to == procurementV1.PurchaseOrder_SUBMITTED {
		if _, err := s.approvalRequestRepo.Create(ctx, &approvalV1.CreateApprovalRequestRequest{
			Data: &approvalV1.ApprovalRequest{
				Title:    trans.Ptr(fmt.Sprintf("采购单 %s", old.GetPoNumber())),
				BizType:  trans.Ptr(poApprovalBizType),
				BizRef:   trans.Ptr(fmt.Sprintf(poApprovalBizRef, id)),
				Summary:  trans.Ptr(fmt.Sprintf("供应商 %s，总额 %d 分，明细 %d 条", old.GetSupplierCode(), old.GetTotalAmount(), len(old.GetItems()))),
			},
		}); err != nil {
			s.log.Errorf("create approval request for purchase order %d failed: %s", id, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// computeOrderAmounts 校验明细并计算金额：明细 amount = 数量×单价（乘法
// 溢出守卫），总额为累加（加法溢出守卫）。数量须为正、单价非负。
func computeOrderAmounts(po *procurementV1.PurchaseOrder) error {
	if po.GetSupplierCode() == "" {
		return procurementV1.ErrorBadRequest("supplier_code is required")
	}
	if len(po.GetItems()) == 0 {
		return procurementV1.ErrorBadRequest("at least one item is required")
	}

	var total int64
	for _, item := range po.GetItems() {
		if item.GetSkuCode() == "" {
			return procurementV1.ErrorBadRequest("item sku_code is required")
		}
		if item.GetQuantity() <= 0 {
			return procurementV1.ErrorBadRequest("item quantity must be positive")
		}
		if item.GetUnitPrice() < 0 {
			return procurementV1.ErrorBadRequest("item unit_price must not be negative")
		}

		amount, overflow := mulChecked(item.GetQuantity(), item.GetUnitPrice())
		if overflow {
			return procurementV1.ErrorBadRequest("item amount overflow")
		}
		item.Amount = trans.Ptr(amount)

		var addOverflow bool
		total, addOverflow = addChecked(total, amount)
		if addOverflow {
			return procurementV1.ErrorBadRequest("total amount overflow")
		}
	}

	po.TotalAmount = trans.Ptr(total)
	return nil
}
