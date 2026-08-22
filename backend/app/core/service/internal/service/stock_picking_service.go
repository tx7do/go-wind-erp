package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

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
func (s *StockPickingService) createInternalPicking(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*emptypb.Empty, error) {
	// 调拨的 source/dest 由仓库推导，但当前设计下客户端需提供 from/to
	// 仓库信息。由于 proto 未显式携带 from/to warehouse code（location
	// 由服务层推导），这里暂要求客户端在 partner_code 或 remark 中不携带
	// 仓库信息——而是由后续 proto 扩展字段补充。当前暂不支持直接创建
	// 调拨拣货单（需先扩展 proto 添加 from/to warehouse 字段）。
	//
	// TODO: 扩展 proto 添加 from_warehouse_code / to_warehouse_code 字段
	// 后启用调拨创建。当前返回 not implemented。
	_ = req
	return nil, inventoryV1.ErrorBadRequest("internal transfer picking creation not yet implemented")
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

		// 创建 move-line（执行记录，借鉴 Odoo stock.move.line._action_done）。
		if err = s.stockMoveLineRepo.CreateTx(ctx, tx, move.GetId(), req.GetId(),
			productCode, sourceLocID, destLocID, executedQty); err != nil {
			return nil, err
		}

		// 更新 source quant：quantity -= executedQty（防负，原子条件更新）。
		// 先按 location+product 取回 quant 行ID，再 ApplyDeltaTx。
		srcQuant, qerr := s.stockQuantRepo.FindByLocationProductTx(ctx, tx, sourceLocID, productCode)
		if qerr != nil {
			err = qerr
			return nil, err
		}
		if _, err = s.stockQuantRepo.ApplyDeltaTx(ctx, tx, srcQuant.GetId(), -executedQty); err != nil {
			return nil, err
		}

		// 更新 dest quant：quantity += executedQty（EnsureForLocation 先确保行存在）。
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
