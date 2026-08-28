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
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

// 销售审批联动约定：审批单 biz_type 固定为 "SALES_ORDER"，
// biz_ref 形如 "SALES_ORDER:{soId}"。审批中心通过/驳回时按此回写销售单状态。
const (
	soApprovalBizType = "SALES_ORDER"
	soApprovalBizRef  = "SALES_ORDER:%d"
)

// SalesOrderService 销售单服务
type SalesOrderService struct {
	salesV1.UnimplementedSalesOrderServiceServer

	log *log.Helper

	salesOrderRepo      *data.SalesOrderRepo
	approvalRequestRepo *data.ApprovalRequestRepo
	receivableRepo      *data.ReceivableRepo
	locationRepo        *data.LocationRepo
	stockPickingRepo    *data.StockPickingRepo
}

func NewSalesOrderService(
	ctx *bootstrap.Context,
	salesOrderRepo *data.SalesOrderRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	receivableRepo *data.ReceivableRepo,
	locationRepo *data.LocationRepo,
	stockPickingRepo *data.StockPickingRepo,
) *SalesOrderService {
	svc := &SalesOrderService{
		log:                 ctx.NewLoggerHelper("sales_order/service/core-service"),
		salesOrderRepo:      salesOrderRepo,
		approvalRequestRepo: approvalRequestRepo,
		receivableRepo:      receivableRepo,
		locationRepo:        locationRepo,
		stockPickingRepo:    stockPickingRepo,
	}

	return svc
}

func (s *SalesOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListSalesOrderResponse, error) {
	return s.salesOrderRepo.List(ctx, req)
}

func (s *SalesOrderService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.CountSalesOrderResponse, error) {
	count, err := s.salesOrderRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &salesV1.CountSalesOrderResponse{Count: uint64(count)}, nil
}

func (s *SalesOrderService) Get(ctx context.Context, req *salesV1.GetSalesOrderRequest) (*salesV1.SalesOrder, error) {
	return s.salesOrderRepo.Get(ctx, req)
}

func (s *SalesOrderService) Create(ctx context.Context, req *salesV1.CreateSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	// 校验明细并预计算金额（乘法溢出守卫 + 累加守卫）。
	if err := computeSalesOrderAmounts(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.salesOrderRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Update 仅 DRAFT 可改（状态守卫）；携带明细时重算金额并整体替换。
func (s *SalesOrderService) Update(ctx context.Context, req *salesV1.UpdateSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	old, err := s.salesOrderRepo.Get(ctx, &salesV1.GetSalesOrderRequest{
		QueryBy: &salesV1.GetSalesOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	// DRAFT 可改；REJECTED 支持驳回后修改重提。
	if old.GetStatus() != salesV1.SalesOrder_DRAFT &&
		old.GetStatus() != salesV1.SalesOrder_REJECTED {
		return nil, salesV1.ErrorConflict("only draft or rejected sales orders can be updated")
	}

	if err := computeSalesOrderAmounts(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.salesOrderRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Delete 仅 DRAFT/CANCELLED 可删（终态审计记录与在途单不可抹除）。
func (s *SalesOrderService) Delete(ctx context.Context, req *salesV1.DeleteSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}

	old, err := s.salesOrderRepo.Get(ctx, &salesV1.GetSalesOrderRequest{
		QueryBy: &salesV1.GetSalesOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	if old.GetStatus() != salesV1.SalesOrder_DRAFT &&
		old.GetStatus() != salesV1.SalesOrder_CANCELLED {
		return nil, salesV1.ErrorConflict("only draft or cancelled sales orders can be deleted")
	}

	if err := s.salesOrderRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Submit 提交：DRAFT→SUBMITTED（原子迁移），并自动生成关联审批请求
// （申请人由审批 repo 从 viewer 推导）。
func (s *SalesOrderService) Submit(ctx context.Context, req *salesV1.SubmitSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), salesV1.SalesOrder_SUBMITTED, true)
}

// Approve 通过：SUBMITTED→APPROVED（管理端直审；审批中心路径经
// approval 同步触发同一迁移）。
func (s *SalesOrderService) Approve(ctx context.Context, req *salesV1.ApproveSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), salesV1.SalesOrder_APPROVED, false)
}

// Reject 驳回：SUBMITTED→REJECTED。
func (s *SalesOrderService) Reject(ctx context.Context, req *salesV1.RejectSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), salesV1.SalesOrder_REJECTED, false)
}

// Complete 完结：APPROVED→COMPLETED（出货完结）。
func (s *SalesOrderService) Complete(ctx context.Context, req *salesV1.CompleteSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}
	return s.transition(ctx, req.GetId(), salesV1.SalesOrder_COMPLETED, false)
}

// Cancel 取消：非终态→CANCELLED，仅发起人本人。
func (s *SalesOrderService) Cancel(ctx context.Context, req *salesV1.CancelSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}

	old, err := s.salesOrderRepo.Get(ctx, &salesV1.GetSalesOrderRequest{
		QueryBy: &salesV1.GetSalesOrderRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	if !validateSalesOrderTransition(old.GetStatus(), salesV1.SalesOrder_CANCELLED) {
		return nil, salesV1.ErrorConflict("sales order status transition not allowed")
	}

	// 仅发起人（created_by）可取消自己的单据。
	caller, ok := approvalViewerUserID(ctx)
	if !ok || caller != old.GetCreatedBy() {
		return nil, salesV1.ErrorConflict("only the creator can cancel this sales order")
	}

	if err := s.salesOrderRepo.TransitionStatus(ctx, req.GetId(), old.GetStatus(), salesV1.SalesOrder_CANCELLED); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// transition 通用原子迁移：读-验-条件更新（0 行→409）。
// createApproval 为 true 时（Submit 路径）在迁移成功后生成审批请求；
// 审批生成失败不影响提交结果，仅记录（审批单可人工补建）。
func (s *SalesOrderService) transition(
	ctx context.Context,
	id uint32,
	to salesV1.SalesOrder_Status,
	createApproval bool,
) (*emptypb.Empty, error) {
	old, err := s.salesOrderRepo.Get(ctx, &salesV1.GetSalesOrderRequest{
		QueryBy: &salesV1.GetSalesOrderRequest_Id{Id: id},
	})
	if err != nil {
		return nil, err
	}

	if !validateSalesOrderTransition(old.GetStatus(), to) {
		return nil, salesV1.ErrorConflict("sales order status transition not allowed")
	}

	// 职责分离：管理端直审不允许发起人自审自（审批中心路径已由审批域守卫）。
	if to == salesV1.SalesOrder_APPROVED || to == salesV1.SalesOrder_REJECTED {
		if caller, ok := approvalViewerUserID(ctx); ok && caller == old.GetCreatedBy() {
			return nil, salesV1.ErrorConflict("creator cannot approve or reject their own sales order")
		}
	}

	if err := s.salesOrderRepo.TransitionStatus(ctx, id, old.GetStatus(), to); err != nil {
		return nil, err
	}

	// 财务联动：销售单获批即生成全额应收单（手工建账之外的自动来源）。
	// 生成失败不影响审批结果，仅记录（可手工补建）。
	if to == salesV1.SalesOrder_APPROVED {
		if _, err := s.receivableRepo.Create(ctx, &financeV1.CreateReceivableRequest{
			Data: &financeV1.Receivable{
				SoRef:        trans.Ptr(fmt.Sprintf(soApprovalBizRef, id)),
				CustomerCode: trans.Ptr(old.GetCustomerCode()),
				Amount:       trans.Ptr(old.GetTotalAmount()),
			},
		}); err != nil {
			s.log.Errorf("create receivable for sales order %d failed: %s", id, err.Error())
		}

		// 库存联动：销售单获批即生成出库拣货单 + 子 moves（借鉴 Odoo
		// _create_picking / _create_stock_moves）。source = SO.warehouse_code
		// 对应仓库的内部位置，dest = 租户客户位置。每个 SO 明细产生一条
		// DRAFT move（sales_order_item_id 指向该明细，借鉴 Odoo sale_line_id）。
		// 生成失败不影响审批结果，仅记录。
		s.createOutboundPicking(ctx, old, id)
	}

	if createApproval && to == salesV1.SalesOrder_SUBMITTED {
		if _, err := s.approvalRequestRepo.Create(ctx, &approvalV1.CreateApprovalRequestRequest{
			Data: &approvalV1.ApprovalRequest{
				Title:   trans.Ptr(fmt.Sprintf("销售单 %s", old.GetSoNumber())),
				BizType: trans.Ptr(soApprovalBizType),
				BizRef:  trans.Ptr(fmt.Sprintf(soApprovalBizRef, id)),
				Summary: trans.Ptr(fmt.Sprintf("客户 %s，总额 %d 分，明细 %d 条", old.GetCustomerCode(), old.GetTotalAmount(), len(old.GetItems()))),
			},
		}); err != nil {
			s.log.Errorf("create approval request for sales order %d failed: %s", id, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// createOutboundPicking 销售单获批后创建出库拣货单 + 子 moves（借鉴 Odoo
// _create_picking / _create_stock_moves）。source = SO.warehouse_code 对应
// 仓库的内部位置，dest = 租户客户虚拟位置。每个 SO 明细产生一条 DRAFT move
// （sales_order_item_id 指向该明细，借鉴 Odoo sale_line_id）。
// 生成失败仅记录日志，不影响审批结果。
func (s *SalesOrderService) createOutboundPicking(
	ctx context.Context,
	so *salesV1.SalesOrder,
	soID uint32,
) {
	warehouseCode := so.GetWarehouseCode()
	if warehouseCode == "" {
		s.log.Errorf("create outbound picking for so %d failed: no warehouse_code", soID)
		return
	}

	// 推导 source（仓库内部位置）与 dest（客户虚拟位置）。
	sourceLocID, err := s.locationRepo.GetLocationID(ctx, warehouseCode)
	if err != nil {
		s.log.Errorf("create outbound picking for so %d failed: resolve source location: %s", soID, err.Error())
		return
	}
	destLocID, err := s.locationRepo.GetCustomerLocationID(ctx)
	if err != nil {
		s.log.Errorf("create outbound picking for so %d failed: resolve customer location: %s", soID, err.Error())
		return
	}

	// 构建子 moves：每条 SO 明细对应一条 DRAFT move。
	moves := make([]*inventoryV1.StockMove, 0, len(so.GetItems()))
	for _, item := range so.GetItems() {
		if item == nil || item.GetId() == 0 {
			continue
		}
		moves = append(moves, &inventoryV1.StockMove{
			ProductCode:      item.SkuCode,
			PlannedQuantity:  item.Quantity,
			SalesOrderItemId: item.Id,
		})
	}
	if len(moves) == 0 {
		s.log.Errorf("create outbound picking for so %d failed: no valid items", soID)
		return
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_OUTGOING.Enum(),
			SourceLocationId:      trans.Ptr(sourceLocID),
			DestinationLocationId: trans.Ptr(destLocID),
			PartnerCode:           so.CustomerCode,
			Moves:                 moves,
		},
	})
	if err != nil {
		s.log.Errorf("create outbound picking for so %d failed: %s", soID, err.Error())
	}
}

// computeSalesOrderAmounts 校验明细并计算金额：明细 amount = 数量×单价（乘法
// 溢出守卫），总额为累加（加法溢出守卫）。数量须为正、单价非负。
func computeSalesOrderAmounts(so *salesV1.SalesOrder) error {
	if so.GetCustomerCode() == "" {
		return salesV1.ErrorBadRequest("customer_code is required")
	}
	if so.GetWarehouseCode() == "" {
		return salesV1.ErrorBadRequest("warehouse_code is required")
	}
	if len(so.GetItems()) == 0 {
		return salesV1.ErrorBadRequest("at least one item is required")
	}

	var total int64
	for _, item := range so.GetItems() {
		if item.GetSkuCode() == "" {
			return salesV1.ErrorBadRequest("item sku_code is required")
		}
		if item.GetQuantity() <= 0 {
			return salesV1.ErrorBadRequest("item quantity must be positive")
		}
		if item.GetUnitPrice() < 0 {
			return salesV1.ErrorBadRequest("item unit_price must not be negative")
		}

		amount, overflow := mulChecked(item.GetQuantity(), item.GetUnitPrice())
		if overflow {
			return salesV1.ErrorBadRequest("item amount overflow")
		}
		item.Amount = trans.Ptr(amount)

		var addOverflow bool
		total, addOverflow = addChecked(total, amount)
		if addOverflow {
			return salesV1.ErrorBadRequest("total amount overflow")
		}
	}

	so.TotalAmount = trans.Ptr(total)
	return nil
}
