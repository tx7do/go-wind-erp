package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/stockmove"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockMoveRepo 库存移动计划仓储（借鉴 Odoo stock.move 的计划角色）。
// 不对外暴露独立 CRUD API——moves 仅作为 picking 的嵌套子记录存在，
// 通过 CreateMovesForPicking 创建、GetMovesByPicking 读取。
type StockMoveRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	stateConverter *mapper.EnumTypeConverter[inventoryV1.StockMove_State, stockmove.State]
}

func NewStockMoveRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StockMoveRepo {
	repo := &StockMoveRepo{
		log:       ctx.NewLoggerHelper("stock_move/repo/core-service"),
		entClient: entClient,
	}
	repo.stateConverter = mapper.NewEnumTypeConverter[inventoryV1.StockMove_State, stockmove.State](inventoryV1.StockMove_State_name, inventoryV1.StockMove_State_value)
	return repo
}

// CreateMovesForPicking 为拣货单批量创建子移动计划（状态恒为 DRAFT）。
// source/dest location 由调用方（StockPickingRepo.Create）从拣货单的
// location 推导后传入——客户端不直接设置 per-move location（借鉴 Odoo
// _create_stock_moves / _prepare_stock_move_vals 的 location 来源）。
func (r *StockMoveRepo) CreateMovesForPicking(
	ctx context.Context,
	pickingID uint32,
	sourceLocationID, destinationLocationID uint32,
	moves []*inventoryV1.StockMove,
) error {
	if pickingID == 0 || len(moves) == 0 {
		return nil
	}

	for _, move := range moves {
		if move == nil {
			continue
		}
		builder := r.entClient.Client().StockMove.Create().
			SetPickingID(pickingID).
			SetSourceLocationID(sourceLocationID).
			SetDestinationLocationID(destinationLocationID).
			SetState(stockmove.StateDraft).
			SetCreatedAt(time.Now())
		if move.ProductCode != nil {
			builder.SetNillableProductCode(move.ProductCode)
		}
		if move.PlannedQuantity != nil {
			builder.SetNillablePlannedQuantity(move.PlannedQuantity)
		}
		if move.PurchaseOrderItemId != nil {
			builder.SetNillablePurchaseOrderItemID(move.PurchaseOrderItemId)
		}
		if move.SalesOrderItemId != nil {
			builder.SetNillableSalesOrderItemID(move.SalesOrderItemId)
		}
		if _, err := builder.Save(ctx); err != nil {
			r.log.Errorf("insert stock_move failed: %s", err.Error())
			return inventoryV1.ErrorInternalServerError("insert stock_move failed")
		}
	}
	return nil
}

// GetMovesByPicking 取拣货单的全部子移动（含状态，用于 Get 组装与 Validate
// 执行）。仅在事务外调用（Validate 用 GetConfirmedMovesByPickingTx）。
func (r *StockMoveRepo) GetMovesByPicking(
	ctx context.Context,
	pickingID uint32,
) []*inventoryV1.StockMove {
	rows, err := r.entClient.Client().StockMove.Query().
		Where(stockmove.PickingIDEQ(pickingID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query stock_moves by picking failed: %s", err.Error())
		return nil
	}
	return r.mapMoveRows(rows)
}

// GetConfirmedMovesByPickingTx 取拣货单中处于 CONFIRMED 状态的子移动（供
// StockPickingService.Validate 在事务内执行）。
func (r *StockMoveRepo) GetConfirmedMovesByPickingTx(
	ctx context.Context,
	tx *ent.Tx,
	pickingID uint32,
) ([]*inventoryV1.StockMove, error) {
	rows, err := tx.StockMove.Query().
		Where(stockmove.PickingIDEQ(pickingID), stockmove.StateEQ(stockmove.StateConfirmed)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query confirmed stock_moves tx failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("query confirmed stock_moves failed")
	}
	return r.mapMoveRows(rows), nil
}

// TransitionMoveStateTx 事务内原子状态迁移：仅当当前状态为 from 时才更新
// 为 to。0 行受影响说明已被并发变更，返回 Conflict。
func (r *StockMoveRepo) TransitionMoveStateTx(
	ctx context.Context,
	tx *ent.Tx,
	id uint32,
	from, to inventoryV1.StockMove_State,
) error {
	fromEntity := *r.stateConverter.ToEntity(trans.Ptr(from))
	toEntity := *r.stateConverter.ToEntity(trans.Ptr(to))

	builder := tx.StockMove.Update().
		Where(stockmove.IDEQ(id)).
		Where(stockmove.StateEQ(fromEntity)).
		SetState(toEntity).
		SetUpdatedAt(time.Now())

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("transition stock_move state failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("transition stock_move state failed")
	}
	if n == 0 {
		return inventoryV1.ErrorConflict("stock_move state changed concurrently")
	}
	return nil
}

// CancelDraftMovesTx 将拣货单内所有 DRAFT move 置为 CANCELLED（事务内批量，
// 供 StockPickingService.Cancel 使用）。
func (r *StockMoveRepo) CancelDraftMovesTx(
	ctx context.Context,
	tx *ent.Tx,
	pickingID uint32,
) error {
	builder := tx.StockMove.Update().
		Where(stockmove.PickingIDEQ(pickingID), stockmove.StateEQ(stockmove.StateDraft)).
		SetState(stockmove.StateCancelled).
		SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel draft stock_moves failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("cancel draft stock_moves failed")
	}
	return nil
}

// CancelConfirmedMovesTx 将拣货单内所有 CONFIRMED move 置为 CANCELLED。
func (r *StockMoveRepo) CancelConfirmedMovesTx(
	ctx context.Context,
	tx *ent.Tx,
	pickingID uint32,
) error {
	builder := tx.StockMove.Update().
		Where(stockmove.PickingIDEQ(pickingID), stockmove.StateEQ(stockmove.StateConfirmed)).
		SetState(stockmove.StateCancelled).
		SetUpdatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("cancel confirmed stock_moves failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("cancel confirmed stock_moves failed")
	}
	return nil
}

// DeleteMovesByPicking 删除拣货单的全部子移动（供 StockPickingRepo.Delete
// 删除拣货单时级联清理）。
func (r *StockMoveRepo) DeleteMovesByPicking(ctx context.Context, pickingID uint32) error {
	_, err := r.entClient.Client().StockMove.Delete().
		Where(stockmove.PickingIDEQ(pickingID)).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete stock_moves by picking failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("delete stock_moves failed")
	}
	return nil
}

// mapMoveRows 将 ent.StockMove 行映射为 DTO。注意 state 经
// stateConverter 转换。
func (r *StockMoveRepo) mapMoveRows(rows []*ent.StockMove) []*inventoryV1.StockMove {
	items := make([]*inventoryV1.StockMove, 0, len(rows))
	for _, row := range rows {
		dto := &inventoryV1.StockMove{
			Id:                    trans.Ptr(row.ID),
			PickingId:             row.PickingID,
			ProductCode:           row.ProductCode,
			SourceLocationId:      row.SourceLocationID,
			DestinationLocationId: row.DestinationLocationID,
			PlannedQuantity:       row.PlannedQuantity,
			State:                 r.stateConverter.ToDTO(row.State),
			PurchaseOrderItemId:   row.PurchaseOrderItemID,
			SalesOrderItemId:      row.SalesOrderItemID,
			Remark:                row.Remark,
		}
		items = append(items, dto)
	}
	return items
}
