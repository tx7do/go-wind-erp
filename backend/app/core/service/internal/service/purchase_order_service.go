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
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// 采购审批联动约定：审批单 biz_type 固定为 "PURCHASE_ORDER"，
// biz_ref 形如 "PURCHASE_ORDER:{poId}"。审批中心通过/驳回时按此回写采购单状态。
const (
	poApprovalBizType = "PURCHASE_ORDER"
	poApprovalBizRef  = "PURCHASE_ORDER:%d"
)

// PurchaseOrderService 采购单服务
type PurchaseOrderService struct {
	procurementV1.UnimplementedPurchaseOrderServiceServer

	log *log.Helper

	purchaseOrderRepo   *data.PurchaseOrderRepo
	approvalRequestRepo *data.ApprovalRequestRepo
	payableRepo         *data.PayableRepo
	billingGuard        *BillingGuard
	stockQuantRepo      *data.StockQuantRepo
	locationRepo        *data.LocationRepo
	stockPickingRepo    *data.StockPickingRepo
}

func NewPurchaseOrderService(
	ctx *bootstrap.Context,
	purchaseOrderRepo *data.PurchaseOrderRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	payableRepo *data.PayableRepo,
	billingGuard *BillingGuard,
	stockQuantRepo *data.StockQuantRepo,
	locationRepo *data.LocationRepo,
	stockPickingRepo *data.StockPickingRepo,
) *PurchaseOrderService {
	svc := &PurchaseOrderService{
		log:                 ctx.NewLoggerHelper("purchase_order/service/core-service"),
		purchaseOrderRepo:   purchaseOrderRepo,
		approvalRequestRepo: approvalRequestRepo,
		payableRepo:         payableRepo,
		billingGuard:      billingGuard,
		stockQuantRepo:      stockQuantRepo,
		locationRepo:        locationRepo,
		stockPickingRepo:    stockPickingRepo,
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

	// 透明定价配额守卫：订阅到期/月单量超限 → 409（数据保留只读）。
	// SAGA 自动补货草稿（CreateReplenishmentDraft）不走守卫——系统自动行为。
	if err := s.billingGuard.EnsureCanCreateOrder(ctx); err != nil {
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

	// 多级审批守卫：审批单在途且未到终级时，禁止管理端直审绕过级进
	// （审批中心逐级推进是唯一通路）。
	if to == procurementV1.PurchaseOrder_APPROVED || to == procurementV1.PurchaseOrder_REJECTED {
		if cur, total, ok, gerr := s.approvalRequestRepo.PendingStepsByBizRef(ctx,
			fmt.Sprintf(poApprovalBizRef, id)); gerr == nil && ok && total > 1 && cur < total {
			return nil, procurementV1.ErrorConflict(
				"多级审批进行中（第 %d/%d 级），请在审批中心逐级审批", cur, total)
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

		// 库存联动：采购单获批即生成入库拣货单 + 子 moves（借鉴 Odoo
		// _create_picking / _create_stock_moves）。source = 租户供应商
		// 位置，dest = PO.warehouse_code 对应仓库的接收位置。每个 PO 明细
		// 产生一条 DRAFT move（purchase_order_item_id 指向该明细，借鉴
		// Odoo purchase_line_id）。生成失败不影响审批结果，仅记录。
		s.createInboundPicking(ctx, old, id)
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

// createInboundPicking 采购单获批后创建入库拣货单 + 子 moves（借鉴 Odoo
// _create_picking / _create_stock_moves）。source = 租户供应商位置，
// dest = PO.warehouse_code 对应仓库的接收位置。每个 PO 明细产生一条
// DRAFT move（purchase_order_item_id 指向该明细，借鉴 Odoo purchase_line_id）。
// 生成失败仅记录日志，不影响审批结果。
func (s *PurchaseOrderService) createInboundPicking(
	ctx context.Context,
	po *procurementV1.PurchaseOrder,
	poID uint32,
) {
	warehouseCode := po.GetWarehouseCode()
	if warehouseCode == "" {
		s.log.Errorf("create inbound picking for po %d failed: no warehouse_code", poID)
		return
	}

	// 推导 source（供应商位置）与 dest（仓库接收位置）。
	sourceLocID, err := s.locationRepo.GetSupplierLocationID(ctx)
	if err != nil {
		s.log.Errorf("create inbound picking for po %d failed: resolve supplier location: %s", poID, err.Error())
		return
	}
	destLocID, err := s.locationRepo.GetLocationID(ctx, warehouseCode)
	if err != nil {
		s.log.Errorf("create inbound picking for po %d failed: resolve receiving location: %s", poID, err.Error())
		return
	}

	// 构建子 moves：每条 PO 明细对应一条 DRAFT move。
	moves := make([]*inventoryV1.StockMove, 0, len(po.GetItems()))
	for _, item := range po.GetItems() {
		if item == nil || item.GetId() == 0 {
			continue
		}
		moves = append(moves, &inventoryV1.StockMove{
			ProductCode:         item.SkuCode,
			PlannedQuantity:     item.Quantity,
			PurchaseOrderItemId: item.Id,
		})
	}
	if len(moves) == 0 {
		s.log.Errorf("create inbound picking for po %d failed: no valid items", poID)
		return
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:          inventoryV1.StockPicking_INCOMING.Enum(),
			SourceLocationId:     trans.Ptr(sourceLocID),
			DestinationLocationId: trans.Ptr(destLocID),
			PurchaseOrderId:      trans.Ptr(poID),
			PartnerCode:          po.SupplierCode,
			Moves:                moves,
		},
	})
	if err != nil {
		s.log.Errorf("create inbound picking for po %d failed: %s", poID, err.Error())
	}
}

// CreateReplenishmentDraft 补货建议获批后自动创建草稿采购单：
// 供应商取该 SKU 最近采购来源（无历史则不建单，返回错误由调用方记录）；
// 数量补到阈值的 2 倍（至少一个阈值批次）。草稿仍需采购员完善后提交。
// 返回创建的采购单（含单号，供调用方通知使用）。
func (s *PurchaseOrderService) CreateReplenishmentDraft(
	ctx context.Context,
	warehouseCode, skuCode string,
) (*procurementV1.PurchaseOrder, error) {
	supplier, err := s.purchaseOrderRepo.LastSupplierForSku(ctx, skuCode)
	if err != nil {
		return nil, err
	}
	if supplier == "" {
		return nil, fmt.Errorf("no supplier history for sku %s", skuCode)
	}

	// 将仓库编码解析为接收位置ID，再按 location+product 查在手量。
	locationID, err := s.locationRepo.GetLocationID(ctx, warehouseCode)
	if err != nil {
		return nil, err
	}
	quant, err := s.stockQuantRepo.FindByLocationProduct(ctx, locationID, skuCode)
	if err != nil {
		return nil, err
	}
	qty := suggestReplenishQty(quant.GetQuantity(), defaultLowStockThreshold)

	po, err := s.purchaseOrderRepo.Create(ctx, &procurementV1.CreatePurchaseOrderRequest{
		Data: &procurementV1.PurchaseOrder{
			SupplierCode:  trans.Ptr(supplier),
			WarehouseCode: trans.Ptr(warehouseCode),
			Remark:        trans.Ptr(fmt.Sprintf("低库存自动补货草稿（%s/%s），请完善后提交", warehouseCode, skuCode)),
			Items: []*procurementV1.PurchaseOrderItem{
				{SkuCode: trans.Ptr(skuCode), Quantity: trans.Ptr(qty), UnitPrice: trans.Ptr(int64(0))},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return po, nil
}

// computeOrderAmounts 校验明细并计算金额：明细 amount = 数量×单价（乘法
// 溢出守卫），总额为累加（加法溢出守卫）。数量须为正、单价非负。
func computeOrderAmounts(po *procurementV1.PurchaseOrder) error {
	if po.GetSupplierCode() == "" {
		return procurementV1.ErrorBadRequest("supplier_code is required")
	}
	if po.GetWarehouseCode() == "" {
		return procurementV1.ErrorBadRequest("warehouse_code is required")
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
