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
	"go-wind-erp/app/core/service/internal/data/ent/supplier"

	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

type SupplierRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[procurementV1.Supplier, ent.Supplier]

	repository *entCrud.Repository[
		ent.SupplierQuery, ent.SupplierSelect,
		ent.SupplierCreate, ent.SupplierCreateBulk,
		ent.SupplierUpdate, ent.SupplierUpdateOne,
		ent.SupplierDelete,
		predicate.Supplier,
		procurementV1.Supplier, ent.Supplier,
	]
}

func NewSupplierRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *SupplierRepo {
	repo := &SupplierRepo{
		log:       ctx.NewLoggerHelper("supplier/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *SupplierRepo) init() {
	r.mapper = mapper.NewCopierMapper[procurementV1.Supplier, ent.Supplier]()

	r.repository = entCrud.NewRepository[
		ent.SupplierQuery, ent.SupplierSelect,
		ent.SupplierCreate, ent.SupplierCreateBulk,
		ent.SupplierUpdate, ent.SupplierUpdateOne,
		ent.SupplierDelete,
		predicate.Supplier,
		procurementV1.Supplier, ent.Supplier,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *SupplierRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Supplier.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, procurementV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *SupplierRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListSupplierResponse, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Supplier.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &procurementV1.ListSupplierResponse{Total: 0, Items: nil}, nil
	}

	return &procurementV1.ListSupplierResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *SupplierRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Supplier.Query().
		Where(supplier.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, procurementV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *SupplierRepo) Get(ctx context.Context, req *procurementV1.GetSupplierRequest) (*procurementV1.Supplier, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Supplier.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *procurementV1.GetSupplierRequest_Id:
		whereCond = append(whereCond, supplier.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *SupplierRepo) Create(ctx context.Context, req *procurementV1.CreateSupplierRequest) (*procurementV1.Supplier, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Supplier.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableCode(req.Data.Code).
		SetNillableName(req.Data.Name).
		SetNillableContact(req.Data.Contact).
		SetNillablePhone(req.Data.Phone).
		SetNillableEnable(req.Data.Enable).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert supplier failed: %s", err.Error())
		return nil, procurementV1.ErrorInternalServerError("insert supplier failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *SupplierRepo) Update(ctx context.Context, req *procurementV1.UpdateSupplierRequest) (*procurementV1.Supplier, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &procurementV1.CreateSupplierRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().Supplier.UpdateOneID(req.GetId())
	builder.Where(supplier.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(supplier.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *procurementV1.Supplier) {
			builder.
				SetNillableCode(req.Data.Code).
				SetNillableName(req.Data.Name).
				SetNillableContact(req.Data.Contact).
				SetNillablePhone(req.Data.Phone).
				SetNillableEnable(req.Data.Enable).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(supplier.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *SupplierRepo) Delete(ctx context.Context, req *procurementV1.DeleteSupplierRequest) error {
	if req == nil {
		return procurementV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().Supplier.Delete()
	delBuilder.Where(supplier.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(supplier.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return procurementV1.ErrorNotFound("supplier not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return procurementV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
