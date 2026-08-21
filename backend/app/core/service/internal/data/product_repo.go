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
	"go-wind-erp/app/core/service/internal/data/ent/product"

	productV1 "go-wind-erp/api/gen/go/product/service/v1"
)

type ProductRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[productV1.Product, ent.Product]

	repository *entCrud.Repository[
		ent.ProductQuery, ent.ProductSelect,
		ent.ProductCreate, ent.ProductCreateBulk,
		ent.ProductUpdate, ent.ProductUpdateOne,
		ent.ProductDelete,
		predicate.Product,
		productV1.Product, ent.Product,
	]
}

func NewProductRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ProductRepo {
	repo := &ProductRepo{
		log:           ctx.NewLoggerHelper("product/repo/core-service"),
		entClient:     entClient,
	}

	repo.init()

	return repo
}

func (r *ProductRepo) init() {
	r.mapper = mapper.NewCopierMapper[productV1.Product, ent.Product]()

	r.repository = entCrud.NewRepository[
		ent.ProductQuery, ent.ProductSelect,
		ent.ProductCreate, ent.ProductCreateBulk,
		ent.ProductUpdate, ent.ProductUpdateOne,
		ent.ProductDelete,
		predicate.Product,
		productV1.Product, ent.Product,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *ProductRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Product.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, productV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *ProductRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*productV1.ListProductResponse, error) {
	if req == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Product.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &productV1.ListProductResponse{Total: 0, Items: nil}, nil
	}

	return &productV1.ListProductResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ProductRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Product.Query().
		Where(product.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, productV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ProductRepo) Get(ctx context.Context, req *productV1.GetProductRequest) (*productV1.Product, error) {
	if req == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Product.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *productV1.GetProductRequest_Id:
		whereCond = append(whereCond, product.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *ProductRepo) Create(ctx context.Context, req *productV1.CreateProductRequest) (*productV1.Product, error) {
	if req == nil || req.Data == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Product.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableCode(req.Data.Code).
		SetNillableName(req.Data.Name).
		SetNillableSpec(req.Data.Spec).
		SetNillableUnit(req.Data.Unit).
		SetNillableEnable(req.Data.Enable).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert product failed: %s", err.Error())
		return nil, productV1.ErrorInternalServerError("insert product failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *ProductRepo) Update(ctx context.Context, req *productV1.UpdateProductRequest) (*productV1.Product, error) {
	if req == nil || req.Data == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	// 如果不存在则创建
	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &productV1.CreateProductRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().Product.UpdateOneID(req.GetId())
	builder.Where(product.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(product.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *productV1.Product) {
			builder.
				SetNillableCode(req.Data.Code).
				SetNillableName(req.Data.Name).
				SetNillableSpec(req.Data.Spec).
				SetNillableUnit(req.Data.Unit).
				SetNillableEnable(req.Data.Enable).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			// updated_by 强制由服务端 viewer context 推导，忽略客户端传入值
			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(product.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *ProductRepo) Delete(ctx context.Context, req *productV1.DeleteProductRequest) error {
	if req == nil {
		return productV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().Product.Delete()
	delBuilder.Where(product.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(product.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return productV1.ErrorNotFound("product not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return productV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
