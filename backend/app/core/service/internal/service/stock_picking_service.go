package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

// StockPickingService 拣货单服务（借鉴 Odoo stock.picking：一等文档，有生命周期）。
// 拣货单状态从子 moves 派生（不存储）。确认（Confirm）将 DRAFT moves 迁至
// CONFIRMED；校验（Validate）在单事务内创建 move-lines、更新 stock_quant、
// 回写采购收货量（借鉴 Odoo button_validate / _action_done）；取消（Cancel）
// 将非终态 moves 迁至 CANCELLED。
type StockPickingService struct {
	inventoryV1.UnimplementedStockPickingServiceServer

	log *log.Helper

	stockPickingRepo  *data.StockPickingRepo
	stockQuantRepo    *data.StockQuantRepo
	stockMoveLineRepo *data.StockMoveLineRepo
	purchaseOrderRepo *data.PurchaseOrderRepo
	salesOrderRepo    *data.SalesOrderRepo
	receivableRepo    *data.ReceivableRepo
	payableRepo       *data.PayableRepo
	locationRepo      *data.LocationRepo
	saga              *procurementSagaSeam
}

func NewStockPickingService(
	ctx *bootstrap.Context,
	stockPickingRepo *data.StockPickingRepo,
	stockQuantRepo *data.StockQuantRepo,
	stockMoveLineRepo *data.StockMoveLineRepo,
	purchaseOrderRepo *data.PurchaseOrderRepo,
	salesOrderRepo *data.SalesOrderRepo,
	receivableRepo *data.ReceivableRepo,
	payableRepo *data.PayableRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	locationRepo *data.LocationRepo,
	messageRepo *data.InternalMessageRepo,
	recipientRepo *data.InternalMessageRecipientRepo,
) *StockPickingService {
	svc := &StockPickingService{
		log:               ctx.NewLoggerHelper("stock_picking/service/core-service"),
		stockPickingRepo:  stockPickingRepo,
		stockQuantRepo:    stockQuantRepo,
		stockMoveLineRepo: stockMoveLineRepo,
		purchaseOrderRepo: purchaseOrderRepo,
		salesOrderRepo:    salesOrderRepo,
		receivableRepo:    receivableRepo,
		payableRepo:       payableRepo,
		locationRepo:      locationRepo,

		saga: &procurementSagaSeam{
			stockQuantRepo:      stockQuantRepo,
			locationRepo:        locationRepo,
			purchaseOrderRepo:   purchaseOrderRepo,
			approvalRequestRepo: approvalRequestRepo,
		},
	}

	_ = messageRepo
	_ = recipientRepo

	return svc
}

func (s *StockPickingService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockPickingResponse, error) {
	return s.stockPickingRepo.List(ctx, req)
}

func (s *StockPickingService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountStockPickingResponse, error) {
	count, err := s.stockPickingRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountStockPickingResponse{Count: uint64(count)}, nil
}

func (s *StockPickingService) Get(ctx context.Context, req *inventoryV1.GetStockPickingRequest) (*inventoryV1.StockPicking, error) {
	return s.stockPickingRepo.Get(ctx, req)
}

// Create 创建拣货单（含嵌套 moves）。type=INTERNAL 调拨时客户端提供明细；
// type=INCOMING 入库仅由采购获批自动创建（客户端不应直接调）。
// source/dest location 由服务层按 type + 仓库推导（客户端不提供）。
func (s *StockPickingService) Create(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	pickingType := req.Data.GetPickingType()

	// source/dest location 由服务层按 type 推导落库（客户端不提供）。
	switch pickingType {
	case inventoryV1.StockPicking_INCOMING:
		// 入库：source = 租户供应商位置，dest = PO.warehouse_code 对应的
		// 接收位置。但入库拣货单仅由采购获批自动创建——客户端直接调
		// 此接口创建入库拣货单不允许（需经采购审批链）。
		return nil, inventoryV1.ErrorBadRequest("incoming pickings are created automatically on PO approval")

	case inventoryV1.StockPicking_OUTGOING:
		// 出库：source = 仓库接收位置，dest = 租户客户位置。出库拣货单
		// 仅由销售单获批自动创建——客户端直接调此接口不允许。
		return nil, inventoryV1.ErrorBadRequest("outgoing pickings are created automatically on SO approval")

	case inventoryV1.StockPicking_INTERNAL:
		// 调拨：source = fromWarehouse 的接收位置，dest = toWarehouse 的
		// 接收位置。客户端在 moves 里提供 per-move 的 product+planned_quantity，
		// 但不提供 per-move location（从 picking 继承）。
		return s.createInternalPicking(ctx, req)

	case inventoryV1.StockPicking_INVENTORY_ADJUSTMENT:
		// 盘点：moves 的 planned_quantity = 带符号差异数（正=盘盈，负=盘亏）。
		// 同一单内必须同号；盘盈 INVENTORY_LOSS→仓库，盘亏 仓库→INVENTORY_LOSS。
		return s.createAdjustmentPicking(ctx, req)

	default:
		return nil, inventoryV1.ErrorBadRequest("unsupported picking type")
	}
}

// createAdjustmentPicking 创建盘点拣货单（INVENTORY_ADJUSTMENT）。
// 借鉴 Odoo InventoryLoss 虚拟位置：盘盈 = 虚拟位置→仓库（入库腿），
// 盘亏 = 仓库→虚拟位置（出库腿，冻结当前均价成本）。同一单必须全正或全负
// （picking 的 source/dest 是单值，混合符号需拆成两张单）。Validate 的
// 双轨 quant 逻辑零改动即可执行（虚拟位置腿自动跳过）。
func (s *StockPickingService) createAdjustmentPicking(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*emptypb.Empty, error) {
	whCode := req.Data.GetFromWarehouseCode()
	if whCode == "" {
		return nil, inventoryV1.ErrorBadRequest("from_warehouse_code is required for inventory adjustment")
	}

	whLocID, err := s.locationRepo.GetLocationID(ctx, whCode)
	if err != nil {
		return nil, err
	}
	lossLocID, err := s.locationRepo.GetInventoryLossLocationID(ctx)
	if err != nil {
		return nil, err
	}

	moves := req.Data.GetMoves()
	if len(moves) == 0 {
		return nil, inventoryV1.ErrorBadRequest("at least one move is required")
	}

	hasPositive, hasNegative := false, false
	for _, m := range moves {
		if m == nil || m.GetProductCode() == "" || m.GetPlannedQuantity() == 0 {
			return nil, inventoryV1.ErrorBadRequest("each move must have a valid product_code and non-zero signed planned_quantity")
		}
		if m.GetPlannedQuantity() > 0 {
			hasPositive = true
		} else {
			hasNegative = true
		}
	}
	if hasPositive && hasNegative {
		return nil, inventoryV1.ErrorBadRequest("an adjustment picking must be all-gain or all-loss; split mixed differences into separate pickings")
	}

	// 盘盈：INVENTORY_LOSS→仓库；盘亏：仓库→INVENTORY_LOSS。
	// move 的 planned_quantity 一律取绝对值（Validate 拒绝非正数量），
	// 方向由 source/dest 表达——与 INCOMING/OUTGOING 的语义一致。
	var srcLocID, dstLocID uint32
	absMoves := make([]*inventoryV1.StockMove, 0, len(moves))
	if hasPositive {
		srcLocID, dstLocID = lossLocID, whLocID
		absMoves = moves
	} else {
		srcLocID, dstLocID = whLocID, lossLocID
		for _, m := range moves {
			absMoves = append(absMoves, &inventoryV1.StockMove{
				ProductCode:         m.ProductCode,
				PlannedQuantity:     trans.Ptr(-m.GetPlannedQuantity()),
				PurchaseOrderItemId: m.PurchaseOrderItemId,
				SalesOrderItemId:    m.SalesOrderItemId,
			})
		}
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_INVENTORY_ADJUSTMENT.Enum(),
			SourceLocationId:      trans.Ptr(srcLocID),
			DestinationLocationId: trans.Ptr(dstLocID),
			Moves:                 absMoves,
			Remark:                req.Data.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateSalesReturn 创建销售退货拣货单：INCOMING（CUSTOMER→SO 仓库位置），
// moves 带 sales_order_item_id；Validate 时负向回写 fulfilled_quantity。
// 仓库/客户编码从 SO 推导，退数 ≤ 已履约数由 repo 原子守卫兜底。
func (s *StockPickingService) CreateSalesReturn(ctx context.Context, req *inventoryV1.CreateSalesReturnRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetSalesOrderId() == 0 || len(req.GetItems()) == 0 {
		return nil, inventoryV1.ErrorBadRequest("sales_order_id and at least one item are required")
	}
	for _, it := range req.GetItems() {
		if it.GetSalesOrderItemId() == 0 || it.GetQuantity() <= 0 {
			return nil, inventoryV1.ErrorBadRequest("each return item must have a valid sales_order_item_id and positive quantity")
		}
	}

	soDTO, err := s.salesOrderRepo.Get(ctx, &salesV1.GetSalesOrderRequest{
		QueryBy: &salesV1.GetSalesOrderRequest_Id{Id: req.GetSalesOrderId()},
	})
	if err != nil {
		return nil, err
	}

	whLocID, err := s.locationRepo.GetLocationID(ctx, soDTO.GetWarehouseCode())
	if err != nil {
		return nil, err
	}
	custLocID, err := s.locationRepo.GetCustomerLocationID(ctx)
	if err != nil {
		return nil, err
	}

	moves := make([]*inventoryV1.StockMove, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		var item *salesV1.SalesOrderItem
		for _, i := range soDTO.GetItems() {
			if i.GetId() == it.GetSalesOrderItemId() {
				item = i
				break
			}
		}
		if item == nil {
			return nil, inventoryV1.ErrorBadRequest("sales order item not found on this order")
		}
		if item.GetFulfilledQuantity() < it.GetQuantity() {
			return nil, inventoryV1.ErrorBadRequest("return quantity exceeds fulfilled quantity")
		}
		moves = append(moves, &inventoryV1.StockMove{
			ProductCode:       trans.Ptr(item.GetSkuCode()),
			PlannedQuantity:   trans.Ptr(it.GetQuantity()),
			SalesOrderItemId:  trans.Ptr(it.GetSalesOrderItemId()),
		})
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_INCOMING.Enum(),
			SourceLocationId:      trans.Ptr(custLocID),
			DestinationLocationId: trans.Ptr(whLocID),
			PartnerCode:           soDTO.CustomerCode,
			Moves:                 moves,
			Remark:                req.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreatePurchaseReturn 创建采购退货拣货单：OUTGOING（PO 仓库位置→SUPPLIER），
// moves 带 purchase_order_item_id；Validate 时负向回写 received_quantity。
func (s *StockPickingService) CreatePurchaseReturn(ctx context.Context, req *inventoryV1.CreatePurchaseReturnRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetPurchaseOrderId() == 0 || len(req.GetItems()) == 0 {
		return nil, inventoryV1.ErrorBadRequest("purchase_order_id and at least one item are required")
	}
	for _, it := range req.GetItems() {
		if it.GetPurchaseOrderItemId() == 0 || it.GetQuantity() <= 0 {
			return nil, inventoryV1.ErrorBadRequest("each return item must have a valid purchase_order_item_id and positive quantity")
		}
	}

	poDTO, err := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
		QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: req.GetPurchaseOrderId()},
	})
	if err != nil {
		return nil, err
	}

	whLocID, err := s.locationRepo.GetLocationID(ctx, poDTO.GetWarehouseCode())
	if err != nil {
		return nil, err
	}
	supLocID, err := s.locationRepo.GetSupplierLocationID(ctx)
	if err != nil {
		return nil, err
	}

	moves := make([]*inventoryV1.StockMove, 0, len(req.GetItems()))
	for _, it := range req.GetItems() {
		var item *procurementV1.PurchaseOrderItem
		for _, i := range poDTO.GetItems() {
			if i.GetId() == it.GetPurchaseOrderItemId() {
				item = i
				break
			}
		}
		if item == nil {
			return nil, inventoryV1.ErrorBadRequest("purchase order item not found on this order")
		}
		if item.GetReceivedQuantity() < it.GetQuantity() {
			return nil, inventoryV1.ErrorBadRequest("return quantity exceeds received quantity")
		}
		moves = append(moves, &inventoryV1.StockMove{
			ProductCode:         trans.Ptr(item.GetSkuCode()),
			PlannedQuantity:     trans.Ptr(it.GetQuantity()),
			PurchaseOrderItemId: trans.Ptr(it.GetPurchaseOrderItemId()),
		})
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_OUTGOING.Enum(),
			SourceLocationId:      trans.Ptr(whLocID),
			DestinationLocationId: trans.Ptr(supLocID),
			PurchaseOrderId:       trans.Ptr(req.GetPurchaseOrderId()),
			PartnerCode:           poDTO.SupplierCode,
			Moves:                 moves,
			Remark:                req.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// createInternalPicking 创建调拨拣货单（INTERNAL：仓库间转移）。
// source/dest location 由服务层按 from/to warehouse code 推导——
// 客户端提供仓库编码，不直接提供 location ID。
func (s *StockPickingService) createInternalPicking(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*emptypb.Empty, error) {
	fromWh := req.Data.GetFromWarehouseCode()
	toWh := req.Data.GetToWarehouseCode()
	if fromWh == "" || toWh == "" {
		return nil, inventoryV1.ErrorBadRequest("from_warehouse_code and to_warehouse_code are required for internal transfer")
	}
	if fromWh == toWh {
		return nil, inventoryV1.ErrorBadRequest("source and destination warehouses must differ")
	}

	// 推导 source/dest location（各自仓库的 receiving_location_id）。
	sourceLocID, err := s.locationRepo.GetLocationID(ctx, fromWh)
	if err != nil {
		return nil, err
	}
	destLocID, err := s.locationRepo.GetLocationID(ctx, toWh)
	if err != nil {
		return nil, err
	}

	// 构建 moves：客户端提供 per-move 的 product+planned_quantity，
	// location 从 picking 继承。
	moves := req.Data.GetMoves()
	if len(moves) == 0 {
		return nil, inventoryV1.ErrorBadRequest("at least one move is required")
	}
	for _, m := range moves {
		if m == nil || m.GetProductCode() == "" || m.GetPlannedQuantity() <= 0 {
			return nil, inventoryV1.ErrorBadRequest("each move must have a valid product_code and positive planned_quantity")
		}
	}

	_, err = s.stockPickingRepo.Create(ctx, &inventoryV1.CreateStockPickingRequest{
		Data: &inventoryV1.StockPicking{
			PickingType:           inventoryV1.StockPicking_INTERNAL.Enum(),
			SourceLocationId:      trans.Ptr(sourceLocID),
			DestinationLocationId: trans.Ptr(destLocID),
			Moves:                 moves,
			Remark:                req.Data.Remark,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Confirm 确认拣货单：将该 picking 下所有 DRAFT move 迁至 CONFIRMED
// （借鉴 Odoo action_confirm）。在事务内批量迁移。
func (s *StockPickingService) Confirm(ctx context.Context, req *inventoryV1.ConfirmStockPickingRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	picking, err := s.stockPickingRepo.Get(ctx, &inventoryV1.GetStockPickingRequest{
		QueryBy: &inventoryV1.GetStockPickingRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	// 仅 DRAFT 态可确认。
	if picking.GetDerivedState() != inventoryV1.StockPicking_DRAFT {
		return nil, inventoryV1.ErrorConflict("picking is not in draft state")
	}

	// 在事务内将所有 DRAFT move 迁至 CONFIRMED。
	tx, terr := s.stockPickingRepo.BeginTx(ctx)
	if terr != nil {
		return nil, terr
	}
	defer func() { s.stockPickingRepo.FinishTx(tx, terr) }()

	for _, move := range picking.GetMoves() {
		if move == nil || move.GetState() != inventoryV1.StockMove_DRAFT {
			continue
		}
		if err := s.stockPickingRepo.TransitionMoveStateTx(ctx, tx, move.GetId(),
			inventoryV1.StockMove_DRAFT, inventoryV1.StockMove_CONFIRMED); err != nil {
			terr = err
			return nil, err
		}
	}

	terr = nil
	return &emptypb.Empty{}, nil
}

// Validate 校验拣货单：在单事务内对每个 CONFIRMED move 创建 move-line、
// 更新 stock_quant（source 减、dest 加）、若入库则回写采购收货量（借鉴
// Odoo button_validate / _action_done）。任一步骤失败 → 整事务回滚，
// 无补偿、无不一致窗口。
func (s *StockPickingService) Validate(ctx context.Context, req *inventoryV1.ValidateStockPickingRequest) (ret *emptypb.Empty, err error) {
	if req == nil || req.GetId() == 0 {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	picking, err := s.stockPickingRepo.Get(ctx, &inventoryV1.GetStockPickingRequest{
		QueryBy: &inventoryV1.GetStockPickingRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	// 仅 CONFIRMED 态可校验。
	if picking.GetDerivedState() != inventoryV1.StockPicking_CONFIRMED {
		return nil, inventoryV1.ErrorConflict("picking is not in confirmed state")
	}

	// 开启事务：全部腿的 quant 回写、move-line 落库、收货量回写均在
	// 事务内执行。任一腿失败则整体回滚，无补偿、无不一致窗口。
	tx, terr := s.stockPickingRepo.BeginTx(ctx)
	if terr != nil {
		return nil, terr
	}

	// 收集出库事件（source 为 INTERNAL 的 move），供事务提交后触发
	// 补货 SAGA 评估。借鉴 Odoo 的 _action_done 后置检查。
	var sagaEvents []stockEvent

	// defer 按 LIFO 顺序执行。先注册的 defer 最后执行。
	// SAGA 分发必须先注册（最后执行），FinishTx 后注册（先执行），
	// 这样事务先提交/回滚，SAGA 再读已提交数据。
	defer func() {
		if err != nil {
			return
		}
		for _, event := range sagaEvents {
			if serr := s.saga.notifyProcurement(ctx, event); serr != nil {
				s.log.Errorf("saga notifyProcurement failed for %s/%s: %s",
					event.warehouseCode, event.skuCode, serr.Error())
			}
		}
	}()
	defer func() { s.stockPickingRepo.FinishTx(tx, err) }()

	// 取该 picking 中所有 CONFIRMED moves（事务内）。
	confirmedMoves, cerr := s.stockPickingRepo.GetConfirmedMovesTx(ctx, tx, req.GetId())
	if cerr != nil {
		err = cerr
		return nil, err
	}

	for _, move := range confirmedMoves {
		if move == nil {
			continue
		}

		sourceLocID := move.GetSourceLocationId()
		destLocID := move.GetDestinationLocationId()
		productCode := move.GetProductCode()
		executedQty := move.GetPlannedQuantity()
		poItemID := move.GetPurchaseOrderItemId()
		soItemID := move.GetSalesOrderItemId()

		if executedQty <= 0 {
			err = inventoryV1.ErrorBadRequest("planned quantity must be positive")
			return nil, err
		}

		// 查 source/dest 的 usage——借鉴 Odoo stock.location.usage：
		// SUPPLIER/CUSTOMER = 虚拟位置（无 quant，跳过该腿的 quant 回写），
		// INTERNAL = 真实位置（有 quant，执行回写）。这是双轨制库存的核心：
		// 只有真实位置之间的移动才同时做 source 减 / dest 加；涉及虚拟
		// 位置的腿跳过 quant 回写（库存从边界 "出现" 或 "消失"）。
		srcUsage, uerr := s.locationRepo.GetUsageTx(ctx, tx, sourceLocID)
		if uerr != nil {
			err = uerr
			return nil, err
		}
		dstUsage, uerr := s.locationRepo.GetUsageTx(ctx, tx, destLocID)
		if uerr != nil {
			err = uerr
			return nil, err
		}

		// 确定 unit_cost：入库腿从采购明细单价取值（quant 加权平均用），
		// 出库/调拨腿从源位置 quant 的 cost_price 冻结（COGS 核算用）。
		// 盘盈/销退回货（source 虚拟 → dest INTERNAL）取 dest quant 当前均价
		// ——按当前均价入库使加权平均保持稳定，避免零成本稀释。
		// 其余虚拟源（无 PO 关联）unit_cost 取 0（仅审计留痕，无财务意义）。
		var unitCost int64
		if srcUsage == inventoryV1.StockLocation_INTERNAL {
			srcQuantForCost, cerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, sourceLocID, productCode)
			if cerr != nil {
				err = cerr
				return nil, err
			}
			unitCost, _ = s.stockQuantRepo.GetCostPriceTx(ctx, tx, srcQuantForCost.GetId())
		} else if poItemID != 0 {
			// 入库：source 是虚拟 SUPPLIER，unit_cost 取采购明细单价。
			unitCost = s.purchaseOrderRepo.GetUnitPriceTx(ctx, tx, poItemID)
		} else if dstUsage == inventoryV1.StockLocation_INTERNAL {
			// 盘盈 / 销售退货回货：source 是 INVENTORY_LOSS/CUSTOMER 虚拟位置，
			// 按 dest 当前均价入库（quant 行不存在则 0）。
			if dstQuantForCost, qerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, destLocID, productCode); qerr == nil {
				unitCost, _ = s.stockQuantRepo.GetCostPriceTx(ctx, tx, dstQuantForCost.GetId())
			}
		}

		// 创建 move-line（执行记录，借鉴 Odoo stock.move.line._action_done）。
		// move-line 始终记录完整的 source→dest 轨迹（含虚拟位置），供审计追溯。
		// unit_cost 冻结执行时的单位成本（入库=采购单价，出库/调拨=源位置 quant
		// 的加权平均成本），供 COGS 核算。
		if err = s.stockMoveLineRepo.CreateTx(ctx, tx, move.GetId(), req.GetId(),
			productCode, sourceLocID, destLocID, executedQty, unitCost); err != nil {
			return nil, err
		}

		// 更新 source quant：quantity -= executedQty（防负，原子条件更新）。
		// 仅对真实位置（INTERNAL）执行——虚拟位置（SUPPLIER）无 quant 行，
		// 跳过（借鉴 Odoo _action_done 对虚拟 source 的处理）。
		if srcUsage == inventoryV1.StockLocation_INTERNAL {
			srcQuant, qerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, sourceLocID, productCode)
			if qerr != nil {
				err = qerr
				return nil, err
			}
			if _, err = s.stockQuantRepo.ApplyDeltaTx(ctx, tx, srcQuant.GetId(), -executedQty); err != nil {
				return nil, err
			}

			// 收集出库事件（source INTERNAL 的扣减），供事务提交后
			// 触发补货 SAGA 评估（借鉴 Odoo _action_done 后置检查）。
			whCode, werr := s.locationRepo.GetWarehouseCodeTx(ctx, tx, sourceLocID)
			if werr != nil {
				err = werr
				return nil, err
			}
			if whCode != "" {
				sagaEvents = append(sagaEvents, stockEvent{
					warehouseCode: whCode,
					skuCode:       productCode,
					delta:         -executedQty,
				})
			}
		}

		// 更新 dest quant：quantity += executedQty（EnsureForLocation 先确保行存在）。
		// 仅对真实位置（INTERNAL）执行——虚拟位置无 quant 行，跳过。
		// 使用 ApplyInboundWithCostTx 同时更新 quantity 和 cost_price（加权平均），
		// unitCost 为上面冻结的源位置成本或采购单价。
		if dstUsage == inventoryV1.StockLocation_INTERNAL {
			if err = s.stockQuantRepo.EnsureForLocationTx(ctx, tx, destLocID, productCode); err != nil {
				return nil, err
			}
			dstQuant, qerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, destLocID, productCode)
			if qerr != nil {
				err = qerr
				return nil, err
			}
			if err = s.stockQuantRepo.ApplyInboundWithCostTx(ctx, tx, dstQuant.GetId(), executedQty, unitCost); err != nil {
				return nil, err
			}
		}

		// move 状态 → DONE（事务内）。
		if err = s.stockPickingRepo.TransitionMoveStateTx(ctx, tx, move.GetId(),
			inventoryV1.StockMove_CONFIRMED, inventoryV1.StockMove_DONE); err != nil {
			return nil, err
		}

		// 入库联动：回写采购明细 received_quantity（借鉴 Odoo 收货回写，
		// 原子条件更新防超收）。若全部明细收齐 → PO 自动完结。
		// 采购退货 = OUTGOING + poItem 关联 → 负向回写（减已收数，减空则 PO 重开）
		// + 应付冲抵（amount -= qty×单价，守卫不冲到已付额之下，尽力而为）。
		if poItemID != 0 {
			if picking.GetPickingType() == inventoryV1.StockPicking_OUTGOING {
				poID, unitPrice, rerr := s.purchaseOrderRepo.ApplyReceiptReturnTx(ctx, tx, poItemID, executedQty)
				if rerr != nil {
					err = rerr
					return nil, err
				}
				if offset, overflow := mulChecked(executedQty, unitPrice); !overflow && offset > 0 {
					if oerr := s.payableRepo.ApplyReturnOffsetTx(ctx, tx,
						fmt.Sprintf("PURCHASE_ORDER:%d", poID), offset); oerr != nil {
						err = oerr
						return nil, err
					}
				}
			} else {
				_, arerr := s.purchaseOrderRepo.ApplyReceiptTx(ctx, tx, poItemID, executedQty)
				if arerr != nil {
					err = arerr
					return nil, err
				}
			}
		}

		// 出库联动：回写销售明细 fulfilled_quantity（镜像采购收货回写，
		// 原子条件更新防超履约）。若全部明细履约齐 → SO 自动完结。
		// 销售退货 = INCOMING + soItem 关联 → 负向回写（减已履约数，减空则 SO 重开）
		// + 应收冲抵（amount -= qty×单价，守卫不冲到已收额之下，尽力而为）。
		if soItemID != 0 {
			if picking.GetPickingType() == inventoryV1.StockPicking_INCOMING {
				soID, unitPrice, rerr := s.salesOrderRepo.ApplyFulfillmentReturnTx(ctx, tx, soItemID, executedQty)
				if rerr != nil {
					err = rerr
					return nil, err
				}
				if offset, overflow := mulChecked(executedQty, unitPrice); !overflow && offset > 0 {
					if oerr := s.receivableRepo.ApplyReturnOffsetTx(ctx, tx,
						fmt.Sprintf("SALES_ORDER:%d", soID), offset); oerr != nil {
						err = oerr
						return nil, err
					}
				}
			} else {
				_, ferr := s.salesOrderRepo.ApplyFulfillmentTx(ctx, tx, soItemID, executedQty)
				if ferr != nil {
					err = ferr
					return nil, err
				}
			}
		}
	}

	// 事务提交（由 FinishTx 执行）。
	ret = &emptypb.Empty{}
	err = nil

	// SAGA 分发由上方先注册的 defer 在 FinishTx 之后执行（读已提交数据）。

	return ret, err
}

// Cancel 取消拣货单：将所有非终态 moves 迁至 CANCELLED（借鉴 Odoo
// action_cancel）。在事务内批量迁移。
func (s *StockPickingService) Cancel(ctx context.Context, req *inventoryV1.CancelStockPickingRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	picking, err := s.stockPickingRepo.Get(ctx, &inventoryV1.GetStockPickingRequest{
		QueryBy: &inventoryV1.GetStockPickingRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	// 终态不可取消。
	if picking.GetDerivedState() == inventoryV1.StockPicking_DONE ||
		picking.GetDerivedState() == inventoryV1.StockPicking_CANCELLED {
		return nil, inventoryV1.ErrorConflict("picking is already in a terminal state")
	}

	// 在事务内将所有非终态 moves 迁至 CANCELLED。
	tx, terr := s.stockPickingRepo.BeginTx(ctx)
	if terr != nil {
		return nil, terr
	}
	defer func() { s.stockPickingRepo.FinishTx(tx, terr) }()

	if err := s.stockPickingRepo.CancelMovesTx(ctx, tx, req.GetId()); err != nil {
		terr = err
		return nil, err
	}

	terr = nil
	return &emptypb.Empty{}, nil
}

func (s *StockPickingService) Delete(ctx context.Context, req *inventoryV1.DeleteStockPickingRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	picking, err := s.stockPickingRepo.Get(ctx, &inventoryV1.GetStockPickingRequest{
		QueryBy: &inventoryV1.GetStockPickingRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}

	// 仅 DRAFT/CANCELLED 可删（终态审计记录与在途单不可抹除）。
	if picking.GetDerivedState() != inventoryV1.StockPicking_DRAFT &&
		picking.GetDerivedState() != inventoryV1.StockPicking_CANCELLED {
		return nil, inventoryV1.ErrorConflict("only draft or cancelled pickings can be deleted")
	}

	if err := s.stockPickingRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
