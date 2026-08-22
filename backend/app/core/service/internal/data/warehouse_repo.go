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
	"go-wind-erp/app/core/service/internal/data/ent/warehouse"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type WarehouseRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[inventoryV1.Warehouse, ent.Warehouse]

	repository *entCrud.Repository[
		ent.WarehouseQuery, ent.WarehouseSelect,
		ent.WarehouseCreate, ent.WarehouseCreateBulk,
		ent.WarehouseUpdate, ent.WarehouseUpdateOne,
		ent.WarehouseDelete,
		predicate.Warehouse,
		inventoryV1.Warehouse, ent.Warehouse,
	]
}

func NewWarehouseRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *WarehouseRepo {
	repo := &WarehouseRepo{
		log:           ctx.NewLoggerHelper("warehouse/repo/core-service"),
		entClient:     entClient,
	}

	repo.init()

	return repo
}

func (r *WarehouseRepo) init() {
	r.mapper = mapper.NewCopierMapper[inventoryV1.Warehouse, ent.Warehouse]()

	r.repository = entCrud.NewRepository[
		ent.WarehouseQuery, ent.WarehouseSelect,
		ent.WarehouseCreate, ent.WarehouseCreateBulk,
		ent.WarehouseUpdate, ent.WarehouseUpdateOne,
		ent.WarehouseDelete,
		predicate.Warehouse,
		inventoryV1.Warehouse, ent.Warehouse,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *WarehouseRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Warehouse.Query()
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

func (r *WarehouseRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListWarehouseResponse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Warehouse.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &inventoryV1.ListWarehouseResponse{Total: 0, Items: nil}, nil
	}

	return &inventoryV1.ListWarehouseResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *WarehouseRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Warehouse.Query().
		Where(warehouse.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, inventoryV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *WarehouseRepo) Get(ctx context.Context, req *inventoryV1.GetWarehouseRequest) (*inventoryV1.Warehouse, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Warehouse.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *inventoryV1.GetWarehouseRequest_Id:
		whereCond = append(whereCond, warehouse.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *WarehouseRepo) Create(ctx context.Context, req *inventoryV1.CreateWarehouseRequest) (*inventoryV1.Warehouse, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

		builder := r.entClient.Client().Warehouse.Create().
			SetNillableTenantID(req.Data.TenantId).
			SetNillableCode(req.Data.Code).
			SetNillableName(req.Data.Name).
			SetNillableLocation(req.Data.Location).
			SetNillableEnable(req.Data.Enable).
			SetNillableReceivingLocationID(req.Data.ReceivingLocationId).
			SetNillableRemark(req.Data.Remark).
			SetNillableCreatedBy(req.Data.CreatedBy).
			SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert warehouse failed: %s", err.Error())
		return nil, inventoryV1.ErrorInternalServerError("insert warehouse failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *WarehouseRepo) Update(ctx context.Context, req *inventoryV1.UpdateWarehouseRequest) (*inventoryV1.Warehouse, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &inventoryV1.CreateWarehouseRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().Warehouse.UpdateOneID(req.GetId())
	builder.Where(warehouse.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(warehouse.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *inventoryV1.Warehouse) {
			builder.
				SetNillableCode(req.Data.Code).
				SetNillableName(req.Data.Name).
				SetNillableLocation(req.Data.Location).
				SetNillableEnable(req.Data.Enable).
				SetNillableReceivingLocationID(req.Data.ReceivingLocationId).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			// updated_by 强制由服务端 viewer context 推导，忽略客户端传入值
			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(warehouse.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *WarehouseRepo) Delete(ctx context.Context, req *inventoryV1.DeleteWarehouseRequest) error {
	if req == nil {
		return inventoryV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().Warehouse.Delete()
	delBuilder.Where(warehouse.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(warehouse.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return inventoryV1.ErrorNotFound("warehouse not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return inventoryV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
