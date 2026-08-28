package data

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/stockquant"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockQuantRepo 库存量仓储（借鉴 Odoo stock.quant）。
// quantity 是库存唯一真相，仅由 StockPickingService.Validate 通过创建
// StockMoveLine 来变更。本仓储提供原子 ApplyDelta（防负）与 EnsureForLocation
// （确保自然键行存在），但不对 API 暴露直接写入口。
type StockQuantRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[inventoryV1.StockQuant, ent.StockQuant]

	repository *entCrud.Repository[
		ent.StockQuantQuery, ent.StockQuantSelect,
		ent.StockQuantCreate, ent.StockQuantCreateBulk,
		ent.StockQuantUpdate, ent.StockQuantUpdateOne,
		ent.StockQuantDelete,
		predicate.StockQuant,
		inventoryV1.StockQuant, ent.StockQuant,
	]
}

func NewStockQuantRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StockQuantRepo {
	repo := &StockQuantRepo{
		log:       ctx.NewLoggerHelper("stock_quant/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

// BeginTx 开启一个 DB 事务，供需跨多表原子执行的场景（如拣货校验）使用。
// 配合 FinishTx 在 defer 中按 err 决定提交/回滚。
func (r *StockQuantRepo) BeginTx(ctx context.Context) (tx *ent.Tx, err error) {
	tx, err = r.entClient.Client().Tx(ctx)
	if err != nil {
		r.log.Errorf("start transaction failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("start transaction failed")
	}
	return tx, nil
}

// FinishTx 根据调用方的命名返回值 err 决定回滚或提交。
func (r *StockQuantRepo) FinishTx(tx *ent.Tx, err error) {
	if tx == nil {
		return
	}
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			r.log.Errorf("transaction rollback failed: %s", rollbackErr.Error())
		}
		return
	}
	if commitErr := tx.Commit(); commitErr != nil {
		r.log.Errorf("transaction commit failed: %s", commitErr.Error())
	}
}

func (r *StockQuantRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.StockQuant, ent.StockQuant]()

	r.repository = entCrud.NewRepository[
		ent.StockQuantQuery, ent.StockQuantSelect,
		ent.StockQuantCreate, ent.StockQuantCreateBulk,
		ent.StockQuantUpdate, ent.StockQuantUpdateOne,
		ent.StockQuantDelete,
		predicate.StockQuant,
		inventoryV1.StockQuant, ent.StockQuant,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *StockQuantRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().StockQuant.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *StockQuantRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockQuantResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockQuant.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListStockQuantResponse{Total: 0, Items: nil}, nil
	}

	return &inventoryV1.ListStockQuantResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *StockQuantRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, inventoryV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *StockQuantRepo) Get(ctx context.Context, req *inventoryV1.GetStockQuantRequest) (*inventoryV1.StockQuant, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockQuant.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetStockQuantRequest_Id:
		whereCond = append(whereCond, stockquant.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// FindByLocationProduct 按 location+product 查库存行（读路径走租户策略）。
func (r *StockQuantRepo) FindByLocationProduct(
	ctx context.Context,
	locationID uint32,
	productCode string,
) (*inventoryV1.StockQuant, error) {
	row, err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.LocationIDEQ(locationID), stockquant.ProductCodeEQ(productCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, inventoryV1.ErrorNotFound("stock_quant not found")
		}
		return nil, inventoryV1.ErrorInternalServerError("query stock_quant failed")
	}
	return r.mapper.ToDTO(row), nil
}

// FindByLocationProductTx 与 FindByLocationProduct 同语义的事务变体，接受显式
// *ent.Tx，使调用方可在单个 DB 事务内运行（如拣货校验）。
func (r *StockQuantRepo) FindByLocationProductTx(
	ctx context.Context,
	tx *ent.Tx,
	locationID uint32,
	productCode string,
) (*inventoryV1.StockQuant, error) {
	row, err := tx.StockQuant.Query().
		Where(stockquant.LocationIDEQ(locationID), stockquant.ProductCodeEQ(productCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, inventoryV1.ErrorNotFound("stock_quant not found")
		}
		return nil, inventoryV1.ErrorInternalServerError("query stock_quant failed")
	}
	return r.mapper.ToDTO(row), nil
}

// EnsureForLocation 入库前确保库存行存在（不存在则以 0 创建；已存在则无动作）。
func (r *StockQuantRepo) EnsureForLocation(
	ctx context.Context,
	locationID uint32,
	productCode string,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	exist, err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.LocationIDEQ(locationID), stockquant.ProductCodeEQ(productCode)).
		Exist(ctx)
	if err != nil {
		return inventoryV1.ErrorInternalServerError("query stock_quant failed")
	}
	if exist {
		return nil
	}
	builder := r.entClient.Client().StockQuant.Create().
		SetNillableLocationID(&locationID).
		SetNillableProductCode(&productCode).
		SetQuantity(0)
	if hasTenant {
		builder.SetTenantID(tid)
	}
	_, err = builder.Save(ctx)
	if err != nil {
		return inventoryV1.ErrorInternalServerError("ensure stock_quant failed")
	}
	return nil
}

// EnsureForLocationTx 与 EnsureForLocation 同语义的事务变体。
func (r *StockQuantRepo) EnsureForLocationTx(
	ctx context.Context,
	tx *ent.Tx,
	locationID uint32,
	productCode string,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	exist, err := tx.StockQuant.Query().
		Where(stockquant.LocationIDEQ(locationID), stockquant.ProductCodeEQ(productCode)).
		Exist(ctx)
	if err != nil {
		return inventoryV1.ErrorInternalServerError("query stock_quant failed")
	}
	if exist {
		return nil
	}
	builder := tx.StockQuant.Create().
		SetNillableLocationID(&locationID).
		SetNillableProductCode(&productCode).
		SetQuantity(0)
	if hasTenant {
		builder.SetTenantID(tid)
	}
	_, err = builder.Save(ctx)
	if err != nil {
		return inventoryV1.ErrorInternalServerError("ensure stock_quant failed")
	}
	return nil
}

// ApplyDelta 原子回写库存数量：quantity = quantity + delta，条件
// quantity + delta >= 0 防负库存（对当前值求值，并发安全）。
// 返回回写后的数量；0 行受影响 → 负库存或行不存在。
func (r *StockQuantRepo) ApplyDelta(
	ctx context.Context,
	id uint32,
	delta int64,
) (int64, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	builder := r.entClient.Client().StockQuant.Update().
		Where(stockquant.IDEQ(id)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d >= 0", s.C(stockquant.FieldQuantity), delta,
			)))
		}).
		AddQuantity(delta)
	if hasTenant {
		builder.Where(stockquant.TenantIDEQ(tid))
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("apply stock_quant delta failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("apply stock_quant delta failed")
	}
	if n == 0 {
		return 0, inventoryV1.ErrorConflict("insufficient stock or stock_quant changed concurrently")
	}

	after, err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return 0, inventoryV1.ErrorInternalServerError("reload stock_quant failed")
	}
	return *after.Quantity, nil
}

// ApplyDeltaTx 与 ApplyDelta 同语义的事务变体。
func (r *StockQuantRepo) ApplyDeltaTx(
	ctx context.Context,
	tx *ent.Tx,
	id uint32,
	delta int64,
) (int64, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	builder := tx.StockQuant.Update().
		Where(stockquant.IDEQ(id)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d >= 0", s.C(stockquant.FieldQuantity), delta,
			)))
		}).
		AddQuantity(delta)
	if hasTenant {
		builder.Where(stockquant.TenantIDEQ(tid))
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("apply stock_quant delta failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("apply stock_quant delta failed")
	}
	if n == 0 {
		return 0, inventoryV1.ErrorConflict("insufficient stock or stock_quant changed concurrently")
	}

	after, err := tx.StockQuant.Query().
		Where(stockquant.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return 0, inventoryV1.ErrorInternalServerError("reload stock_quant failed")
	}
	return *after.Quantity, nil
}

// GetCostPriceTx 在事务内读取某 quant 的当前加权平均成本（出库时冻结
// 到 stock_move_line.unit_cost 用于 COGS 核算）。
func (r *StockQuantRepo) GetCostPriceTx(
	ctx context.Context,
	tx *ent.Tx,
	quantID uint32,
) (int64, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	q := tx.StockQuant.Query().
		Where(stockquant.IDEQ(quantID))
	if hasTenant {
		q.Where(stockquant.TenantIDEQ(tid))
	}
	quant, err := q.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, inventoryV1.ErrorNotFound("stock_quant not found")
		}
		return 0, inventoryV1.ErrorInternalServerError("query stock_quant failed")
	}
	if quant.CostPrice == nil {
		return 0, nil
	}
	return *quant.CostPrice, nil
}

// ApplyInboundWithCostTx 入库腿的原子回写：同时更新 quantity（+=delta）
// 和 cost_price（加权平均），在单条 UPDATE 内完成。防负守卫与 ApplyDeltaTx
// 相同（quantity + delta >= 0）。加权平均公式：
//   new_cost = (old_qty * old_cost + delta_qty * unit_cost) / (old_qty + delta_qty)
// 当 old_qty + delta_qty = 0（清仓后重新入库）时重置为 unit_cost。
func (r *StockQuantRepo) ApplyInboundWithCostTx(
	ctx context.Context,
	tx *ent.Tx,
	quantID uint32,
	qtyDelta int64,
	unitCost int64,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	builder := tx.StockQuant.Update().
		Where(stockquant.IDEQ(quantID)).
		// 防负：quantity + delta >= 0
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d >= 0", s.C(stockquant.FieldQuantity), qtyDelta,
			)))
		}).
		AddQuantity(qtyDelta)
	if hasTenant {
		builder.Where(stockquant.TenantIDEQ(tid))
	}

	// 加权平均成本：在 SQL 层用 CASE 表达式推导。当新总量为 0 时直接取
	// unit_cost（避免除零），否则取加权平均。
	builder.Modify(func(u *sql.UpdateBuilder) {
		u.Set(
			stockquant.FieldCostPrice,
			sql.Expr(fmt.Sprintf(
				"CASE WHEN %s + %d = 0 THEN %d "+
					"ELSE (%s * %s + %d * %d) / (%s + %d) END",
				stockquant.FieldQuantity, qtyDelta, unitCost,
				stockquant.FieldQuantity, stockquant.FieldCostPrice, qtyDelta, unitCost,
				stockquant.FieldQuantity, qtyDelta,
			)),
		)
	})

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("apply inbound with cost failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("apply inbound with cost failed")
	}
	if n == 0 {
		return inventoryV1.ErrorConflict("insufficient stock or stock_quant changed concurrently")
	}
	return nil
}

// SumQuantity 库存总量（所有记录 quantity 之和）。
// 无 GROUP BY 的 SUM 在空集（或全 NULL 列）时返回一行 NULL 而非零行，
// 直接扫 []int64 会被 database/sql 拒绝导致空库看板 500；用 COALESCE
// 在 SQL 层保证非 NULL（ent ScanSlice 不支持 []NullInt64 结构体切片）。
func (r *StockQuantRepo) SumQuantity(ctx context.Context) (int64, error) {
	var totals []int64
	if err := r.entClient.Client().StockQuant.Query().
		Modify(func(se *sql.Selector) {
			se.Select("COALESCE(" + sql.Sum(se.C(stockquant.FieldQuantity)) + ", 0)")
		}).
		Scan(ctx, &totals); err != nil {
		r.log.Errorf("sum quantity failed: %s", err.Error())
		return 0, inventoryV1.ErrorInternalServerError("sum quantity failed")
	}
	if len(totals) == 0 {
		return 0, nil
	}
	return totals[0], nil
}

// CountDistinctSku 在库 SKU 数（按 product_code 去重计数）。
// product_code 可空，NULL 组会使 []string 扫描失败，先过滤空值。
func (r *StockQuantRepo) CountDistinctSku(ctx context.Context) (int, error) {
	var distinctSkus []string
	if err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.ProductCodeNotNil()).
		GroupBy(stockquant.FieldProductCode).
		Scan(ctx, &distinctSkus); err != nil {
		r.log.Errorf("count distinct sku failed: %s", err.Error())
			return 0, inventoryV1.ErrorInternalServerError("count distinct sku failed")
	}
	return len(distinctSkus), nil
}

// ListLowStock 低库存清单：quantity < threshold，按数量升序，最多 limit 条。
func (r *StockQuantRepo) ListLowStock(ctx context.Context, threshold int64, limit int) ([]*inventoryV1.StockQuant, error) {
	rows, err := r.entClient.Client().StockQuant.Query().
		Where(stockquant.QuantityLT(threshold)).
		Order(ent.Asc(stockquant.FieldQuantity)).
		Limit(limit).
		All(ctx)
	if err != nil {
		r.log.Errorf("list low stock failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("list low stock failed")
	}

	items := make([]*inventoryV1.StockQuant, 0, len(rows))
	for _, row := range rows {
		items = append(items, r.mapper.ToDTO(row))
	}
	return items, nil
}
