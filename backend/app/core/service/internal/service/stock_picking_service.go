package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
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
	locationRepo      *data.LocationRepo
	saga              *procurementSagaSeam
}

func NewStockPickingService(
	ctx *bootstrap.Context,
	stockPickingRepo *data.StockPickingRepo,
	stockQuantRepo *data.StockQuantRepo,
	stockMoveLineRepo *data.StockMoveLineRepo,
	purchaseOrderRepo *data.PurchaseOrderRepo,
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

	case inventoryV1.StockPicking_INTERNAL:
		// 调拨：source = fromWarehouse 的接收位置，dest = toWarehouse 的
		// 接收位置。客户端在 moves 里提供 per-move 的 product+planned_quantity，
		// 但不提供 per-move location（从 picking 继承）。
		return s.createInternalPicking(ctx, req)

	default:
		return nil, inventoryV1.ErrorBadRequest("unsupported picking type")
	}
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
	defer func() { s.stockPickingRepo.FinishTx(tx, err) }()

	// 取该 picking 中所有 CONFIRMED moves（事务内）。
	confirmedMoves, cerr := s.stockPickingRepo.GetConfirmedMovesTx(ctx, tx, req.GetId())
	if cerr != nil {
		err = cerr
		return nil, err
	}

	// 收集出库事件（source 为 INTERNAL 的 move），供事务提交后触发
	// 补货 SAGA 评估。借鉴 Odoo 的 _action_done 后置检查。
	var sagaEvents []stockEvent

	for _, move := range confirmedMoves {
		if move == nil {
			continue
		}

		sourceLocID := move.GetSourceLocationId()
		destLocID := move.GetDestinationLocationId()
		productCode := move.GetProductCode()
		executedQty := move.GetPlannedQuantity()
		poItemID := move.GetPurchaseOrderItemId()

		if executedQty <= 0 {
			err = inventoryV1.ErrorBadRequest("planned quantity must be positive")
			return nil, err
		}

		// 查 source/dest 的 usage——借鉴 Odoo stock.location.usage：
		// SUPPLIER = 虚拟位置（无 quant，跳过该腿的 quant 回写），
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

		// 创建 move-line（执行记录，借鉴 Odoo stock.move.line._action_done）。
		// move-line 始终记录完整的 source→dest 轨迹（含虚拟位置），供审计追溯。
		if err = s.stockMoveLineRepo.CreateTx(ctx, tx, move.GetId(), req.GetId(),
			productCode, sourceLocID, destLocID, executedQty); err != nil {
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
		if dstUsage == inventoryV1.StockLocation_INTERNAL {
			if err = s.stockQuantRepo.EnsureForLocationTx(ctx, tx, destLocID, productCode); err != nil {
				return nil, err
			}
			dstQuant, qerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, destLocID, productCode)
			if qerr != nil {
				err = qerr
				return nil, err
			}
			if _, err = s.stockQuantRepo.ApplyDeltaTx(ctx, tx, dstQuant.GetId(), executedQty); err != nil {
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
		if poItemID != 0 {
			_, arerr := s.purchaseOrderRepo.ApplyReceiptTx(ctx, tx, poItemID, executedQty)
			if arerr != nil {
				err = arerr
				return nil, err
			}
		}
	}

	// 事务提交（由 FinishTx 执行）。
	ret = &emptypb.Empty{}
	err = nil

	// 事务提交后触发补货 SAGA 评估（借鉴 Odoo _action_done 后置检查）。
	// 评估失败仅记录，不阻塞出库——下次任何出库会重新评估，最终一致
	// 靠"重评估"而非"重放"（见 procurementSagaSeam 文档）。
	for _, event := range sagaEvents {
		if serr := s.saga.notifyProcurement(ctx, event); serr != nil {
			s.log.Errorf("saga notifyProcurement failed for %s/%s: %s",
				event.warehouseCode, event.skuCode, serr.Error())
		}
	}

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
