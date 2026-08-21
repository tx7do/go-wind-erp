package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/stockmovement"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type StockMovementRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper             *mapper.CopierMapper[inventoryV1.StockMovement, ent.StockMovement]
	movementTypeConverter *mapper.EnumTypeConverter[inventoryV1.StockMovement_MovementType, stockmovement.MovementType]

	repository *entCrud.Repository[
		ent.StockMovementQuery, ent.StockMovementSelect,
		ent.StockMovementCreate, ent.StockMovementCreateBulk,
		ent.StockMovementUpdate, ent.StockMovementUpdateOne,
		ent.StockMovementDelete,
		predicate.StockMovement,
		inventoryV1.StockMovement, ent.StockMovement,
	]
}

func NewStockMovementRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *StockMovementRepo {
	repo := &StockMovementRepo{
		log:           ctx.NewLoggerHelper("stock_movement/repo/core-service"),
		entClient:     entClient,
	}

	repo.init()

	return repo
}

func (r *StockMovementRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.StockMovement, ent.StockMovement]()
	r.movementTypeConverter = mapper.NewEnumTypeConverter[inventoryV1.StockMovement_MovementType, stockmovement.MovementType](inventoryV1.StockMovement_MovementType_name, inventoryV1.StockMovement_MovementType_value)

	r.repository = entCrud.NewRepository[
		ent.StockMovementQuery, ent.StockMovementSelect,
		ent.StockMovementCreate, ent.StockMovementCreateBulk,
		ent.StockMovementUpdate, ent.StockMovementUpdateOne,
		ent.StockMovementDelete,
		predicate.StockMovement,
		inventoryV1.StockMovement, ent.StockMovement,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.movementTypeConverter.NewConverterPair())
}

func (r *StockMovementRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().StockMovement.Query()
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

// SeedDemoMovements 仅由演示数据服务（ERP_DEMO_SEED=true）调用。为给定租户
// 的每对（仓库, SKU）在近 30 日生成若干条库存流水，流水日期沿日轴分布，使
// 看板折线图（MovementTrend）有可见波形。直接走 ent CreateBulk 显式置
// created_at，绕过 StockMovementService.Create 的库存回写/SAGA/收货副作用
// ——这是纯报表演示数据，不应改动真实库存。读取按 TenantPrivacy 隔离，
// tenant_id 由 viewer 上下文提供，但仍在此显式 Set 以双保险。
func (r *StockMovementRepo) SeedDemoMovements(ctx context.Context, tenantID uint32) error {
	warehouses := []string{"WH-01", "WH-02", "WH-03"}
	skus := []string{"SKU-0001", "SKU-0002", "SKU-0003", "SKU-0004", "SKU-0005"}

	const horizon = 30
	now := time.Now()
	type seed struct {
		day  time.Time
		wh   string
		sku  string
		dlt  int64
	}
	var seeds []seed
	for d := horizon - 1; d >= 0; d-- {
		ts := now.AddDate(0, 0, -d)
		// 每天 2~4 条，分散到不同仓库/SKU
		n := 2 + (d % 3)
		for i := 0; i < n; i++ {
			wh := warehouses[(d+i)%len(warehouses)]
			sku := skus[(d+i+1)%len(skus)]
			delta := int64((i + 1))
			if d%2 == 0 {
				delta = -delta
			}
			seeds = append(seeds, seed{day: ts, wh: wh, sku: sku, dlt: delta})
		}
	}

	bulk := r.entClient.Client().StockMovement.MapCreateBulk(seeds, func(b *ent.StockMovementCreate, i int) {
		_ = b.SetCreatedAt(seeds[i].day)
		_ = b.SetTenantID(tenantID)
		_ = b.SetWarehouseCode(seeds[i].wh)
		_ = b.SetSkuCode(seeds[i].sku)
		_ = b.SetDelta(seeds[i].dlt)
		_ = b.SetMovementType(stockmovement.MovementTypeInbound)
		_ = b.SetQuantityBefore(0)
		_ = b.SetQuantityAfter(0)
	})
	if _, err := bulk.Save(ctx); err != nil {
		r.log.Errorf("demo seed movements bulk save failed: %s", err.Error())
		return inventoryV1.ErrorInternalServerError("demo seed movements failed")
	}
	return nil
}

// MovementTrend 返回近 30 日每日库存流水条数。
//
// SQL 层按真实列 created_at 分组计 COUNT(*)，扫描后在 Go 侧把每行的
// created_at 归一到 YYYY-MM-DD 并补齐 30 日全序列（无流水的日期补 0），
// 保证前端折线图轴完整。按真实列分组而非计算列，与 AgingReport 同模式
// （ent GroupBy 只接受真实列），日期截断在客户端完成避免方言差异。
// 读取经 TenantPrivacy 自动按调用者租户隔离，同其余读路径。
func (r *StockMovementRepo) MovementTrend(ctx context.Context) ([]*inventoryV1.MovementTrendPoint, error) {
	type trendRow struct {
		CreatedAt *time.Time `sql:"created_at"`
		Count     int64      `sql:"count"`
	}
	var rows []trendRow

	if err := r.entClient.Client().StockMovement.Query().
		GroupBy(stockmovement.FieldCreatedAt).
		Aggregate(ent.Count()).
		Scan(ctx, &rows); err != nil {
		r.log.Errorf("movement trend query failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("movement trend query failed")
	}

	// 补齐近 30 日全序列：无流水的日期补 0。
	const horizon = 30
	now := time.Now()
	byDay := make(map[string]int64, len(rows))
	for _, row := range rows {
		if row.CreatedAt == nil {
			continue
		}
		byDay[row.CreatedAt.Format("2006-01-02")] = row.Count
	}
	points := make([]*inventoryV1.MovementTrendPoint, 0, horizon)
	for i := horizon - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		points = append(points, &inventoryV1.MovementTrendPoint{
			Date:  day,
			Count: byDay[day],
		})
	}
	return points, nil
}

func (r *StockMovementRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockMovementResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockMovement.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListStockMovementResponse{Total: 0, Items: nil}, nil
	}

	return &inventoryV1.ListStockMovementResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *StockMovementRepo) Get(ctx context.Context, req *inventoryV1.GetStockMovementRequest) (*inventoryV1.StockMovement, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockMovement.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetStockMovementRequest_Id:
		whereCond = append(whereCond, stockmovement.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *StockMovementRepo) Create(ctx context.Context, req *inventoryV1.CreateStockMovementRequest) (*inventoryV1.StockMovement, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockMovement.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillableSkuCode(req.Data.SkuCode).
		SetNillableDelta(req.Data.Delta).
		SetNillableMovementType(r.movementTypeConverter.ToEntity(req.Data.MovementType)).
		SetNillableQuantityBefore(req.Data.QuantityBefore).
		SetNillableQuantityAfter(req.Data.QuantityAfter).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert stock_movement failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert stock_movement failed")
	}

	return r.mapper.ToDTO(t), nil
}

// CreateTx 与 Create 同语义的事务变体，接受显式 *ent.Tx，使调用方可在
// 单个 DB 事务内运行（如调拨双腿）。
func (r *StockMovementRepo) CreateTx(ctx context.Context, tx *ent.Tx, req *inventoryV1.CreateStockMovementRequest) (*inventoryV1.StockMovement, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := tx.StockMovement.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillableSkuCode(req.Data.SkuCode).
		SetNillableDelta(req.Data.Delta).
		SetNillableMovementType(r.movementTypeConverter.ToEntity(req.Data.MovementType)).
		SetNillableQuantityBefore(req.Data.QuantityBefore).
		SetNillableQuantityAfter(req.Data.QuantityAfter).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert stock_movement failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert stock_movement failed")
	}

	return r.mapper.ToDTO(t), nil
}

// HasReversalMarker 是否已存在携带指定冲正标记的流水（冲正幂等检查）。
func (r *StockMovementRepo) HasReversalMarker(ctx context.Context, warehouseCode, skuCode, marker string) (bool, error) {
	return r.entClient.Client().StockMovement.Query().
		Where(stockmovement.WarehouseCodeEQ(warehouseCode)).
		Where(stockmovement.SkuCodeEQ(skuCode)).
		Where(stockmovement.RemarkContains(marker)).
		Exist(ctx)
}


func (r *StockMovementRepo) Delete(ctx context.Context, req *inventoryV1.DeleteStockMovementRequest) error {
	if req == nil {
		return inventoryV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().StockMovement.Delete()
	delBuilder.Where(stockmovement.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(stockmovement.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return inventoryV1.ErrorNotFound("stock_movement not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return inventoryV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
