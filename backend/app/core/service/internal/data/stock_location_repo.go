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
	"go-wind-erp/app/core/service/internal/data/ent/stocklocation"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// LocationRepo 库存位置仓储（借鉴 Odoo stock.location）。
type LocationRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[inventoryV1.StockLocation, ent.StockLocation]

	repository *entCrud.Repository[
		ent.StockLocationQuery, ent.StockLocationSelect,
		ent.StockLocationCreate, ent.StockLocationCreateBulk,
		ent.StockLocationUpdate, ent.StockLocationUpdateOne,
		ent.StockLocationDelete,
		predicate.StockLocation,
		inventoryV1.StockLocation, ent.StockLocation,
	]
}

func NewLocationRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *LocationRepo {
	repo := &LocationRepo{
		log:       ctx.NewLoggerHelper("stock_location/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *LocationRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.StockLocation, ent.StockLocation]()

	r.repository = entCrud.NewRepository[
		ent.StockLocationQuery, ent.StockLocationSelect,
		ent.StockLocationCreate, ent.StockLocationCreateBulk,
		ent.StockLocationUpdate, ent.StockLocationUpdateOne,
		ent.StockLocationDelete,
		predicate.StockLocation,
		inventoryV1.StockLocation, ent.StockLocation,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *LocationRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().StockLocation.Query()
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

func (r *LocationRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListLocationResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockLocation.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListLocationResponse{Total: 0, Items: nil}, nil
	}

	return &inventoryV1.ListLocationResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *LocationRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().StockLocation.Query().
		Where(stocklocation.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, inventoryV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *LocationRepo) Get(ctx context.Context, req *inventoryV1.GetLocationRequest) (*inventoryV1.StockLocation, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockLocation.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetLocationRequest_Id:
		whereCond = append(whereCond, stocklocation.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *LocationRepo) Create(ctx context.Context, req *inventoryV1.CreateLocationRequest) (*inventoryV1.StockLocation, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().StockLocation.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableName(req.Data.Name).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert stock_location failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert stock_location failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *LocationRepo) Update(ctx context.Context, req *inventoryV1.UpdateLocationRequest) (*inventoryV1.StockLocation, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &inventoryV1.CreateLocationRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().StockLocation.UpdateOneID(req.GetId())
	builder.Where(stocklocation.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(stocklocation.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *inventoryV1.StockLocation) {
			builder.
				SetNillableName(req.Data.Name).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(stocklocation.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *LocationRepo) Delete(ctx context.Context, req *inventoryV1.DeleteLocationRequest) error {
	if req == nil {
		return inventoryV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().StockLocation.Delete()
	delBuilder.Where(stocklocation.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(stocklocation.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return inventoryV1.ErrorNotFound("stock_location not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return inventoryV1.ErrorInternalServerError("delete failed")
	}

	return nil
}

// GetLocationID 取仓库的接收位置ID（仓库创建时自动生成的 INTERNAL 位置）。
// 不返回位置对象本身，只返回 ID——调用方只需要 ID 来设置 move/picking 的
// destination_location_id。
func (r *LocationRepo) GetLocationID(ctx context.Context, warehouseCode string) (uint32, error) {
	loc, err := r.entClient.Client().StockLocation.Query().
		Where(stocklocation.WarehouseCodeEQ(warehouseCode), stocklocation.UsageEQ(stocklocation.UsageInternal)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, inventoryV1.ErrorNotFound("receiving location not found for warehouse")
		}
		return 0, inventoryV1.ErrorInternalServerError("query receiving location failed")
	}
	return loc.ID, nil
}

// GetSupplierLocationID 取租户的供应商位置ID（入库 move 的 source location）。
// 每租户仅一条 usage=SUPPLIER 的位置。
func (r *LocationRepo) GetSupplierLocationID(ctx context.Context) (uint32, error) {
	loc, err := r.entClient.Client().StockLocation.Query().
		Where(stocklocation.UsageEQ(stocklocation.UsageSupplier)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, inventoryV1.ErrorNotFound("supplier location not found for tenant")
		}
		return 0, inventoryV1.ErrorInternalServerError("query supplier location failed")
	}
	return loc.ID, nil
}

// GetUsageTx 在事务内查位置的 usage 值。Validate 用此判断 source/dest
// 是否为虚拟位置（SUPPLIER = 虚拟，无 quant，跳过该腿的 quant 回写）。
func (r *LocationRepo) GetUsageTx(
	ctx context.Context,
	tx *ent.Tx,
	locationID uint32,
) (inventoryV1.StockLocation_Usage, error) {
	loc, err := tx.StockLocation.Query().
		Where(stocklocation.IDEQ(locationID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, inventoryV1.ErrorNotFound("location not found")
		}
		return 0, inventoryV1.ErrorInternalServerError("query location failed")
	}
	if loc.Usage == nil {
		return inventoryV1.StockLocation_INTERNAL, nil
	}
	switch *loc.Usage {
	case stocklocation.UsageSupplier:
		return inventoryV1.StockLocation_SUPPLIER, nil
	case stocklocation.UsageInternal:
		return inventoryV1.StockLocation_INTERNAL, nil
	default:
		return inventoryV1.StockLocation_INTERNAL, nil
	}
}

