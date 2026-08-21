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
