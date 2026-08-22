package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/stockmoveline"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockMoveLineRepo 库存移动执行记录仓储（借鉴 Odoo stock.move.line 的执行
// 角色）。这是唯一能变更 StockQuant.quantity 的东西——在
// StockPickingService.Validate 的事务内创建。append-only：仅 CreatedAt。
type StockMoveLineRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewStockMoveLineRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StockMoveLineRepo {
	repo := &StockMoveLineRepo{
		log:       ctx.NewLoggerHelper("stock_move_line/repo/core-service"),
		entClient: entClient,
	}
	return repo
}

// CountAll 统计执行记录总条数（用于看板 movement_count 指标）。
func (r *StockMoveLineRepo) CountAll(ctx context.Context) (int64, error) {
	count, err := r.entClient.Client().StockMoveLine.Query().Count(ctx)
	if err != nil {
		r.log.Errorf("count stock_move_lines failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("count stock_move_lines failed")
	}
	return int64(count), nil
}

// CreateTx 在事务内创建一条执行记录（借鉴 Odoo stock.move.line._action_done
// 的落库动作）。这是唯一能变更 StockQuant.quantity 的路径——调用方
// （StockPickingService.Validate）在同事务内紧接着调 ApplyDeltaTx 改 quant。
func (r *StockMoveLineRepo) CreateTx(
	ctx context.Context,
	tx *ent.Tx,
	moveID uint32,
	pickingID uint32,
	productCode string,
	sourceLocationID uint32,
	destinationLocationID uint32,
	executedQuantity int64,
) error {
	builder := tx.StockMoveLine.Create().
		SetMoveID(moveID).
		SetPickingID(pickingID).
		SetProductCode(productCode).
		SetSourceLocationID(sourceLocationID).
		SetDestinationLocationID(destinationLocationID).
		SetExecutedQuantity(executedQuantity).
		SetCreatedAt(time.Now())

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert stock_move_line failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("insert stock_move_line failed")
	}
	return nil
}

// MovementTrend 近 30 日每日执行记录条数（看板折线图）。按 DATE(created_at)
// 分组计数，无记录的日期补 0。
func (r *StockMoveLineRepo) MovementTrend(ctx context.Context) ([]*inventoryV1.MovementTrendPoint, error) {
	var rows []struct {
		Date  string `sql:"date"`
		Count int64  `sql:"count"`
	}
	err := r.entClient.Client().StockMoveLine.Query().
		Modify(func(s *sql.Selector) {
			s.Where(sql.ExprP("created_at >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)"))
			s.GroupBy("DATE(" + s.C(stockmoveline.FieldCreatedAt) + ")")
			s.Select("DATE(" + s.C(stockmoveline.FieldCreatedAt) + ") as date", "COUNT(*) as count")
		}).
		Scan(ctx, &rows)
	if err != nil {
		r.log.Errorf("movement trend query failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("movement trend query failed")
	}

	// 将查询结果映射到 30 天的完整日期序列，无记录的日期补 0。
	countByDate := make(map[string]int64, len(rows))
	for _, row := range rows {
		countByDate[row.Date] = row.Count
	}

	now := time.Now()
	points := make([]*inventoryV1.MovementTrendPoint, 30)
	for i := 29; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		count := countByDate[date]
		points[29-i] = &inventoryV1.MovementTrendPoint{
			Date:  date,
			Count: count,
		}
	}
	return points, nil
}
