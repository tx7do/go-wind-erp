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
	"go-wind-erp/app/core/service/internal/data/ent/customer"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"

	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

type CustomerRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[salesV1.Customer, ent.Customer]

	repository *entCrud.Repository[
		ent.CustomerQuery, ent.CustomerSelect,
		ent.CustomerCreate, ent.CustomerCreateBulk,
		ent.CustomerUpdate, ent.CustomerUpdateOne,
		ent.CustomerDelete,
		predicate.Customer,
		salesV1.Customer, ent.Customer,
	]
}

func NewCustomerRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *CustomerRepo {
	repo := &CustomerRepo{
		log:       ctx.NewLoggerHelper("customer/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *CustomerRepo) init() {
	r.mapper = mapper.NewCopierMapper[salesV1.Customer, ent.Customer]()

	r.repository = entCrud.NewRepository[
		ent.CustomerQuery, ent.CustomerSelect,
		ent.CustomerCreate, ent.CustomerCreateBulk,
		ent.CustomerUpdate, ent.CustomerUpdateOne,
		ent.CustomerDelete,
		predicate.Customer,
		salesV1.Customer, ent.Customer,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
}

func (r *CustomerRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Customer.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, salesV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *CustomerRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListCustomerResponse, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Customer.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &salesV1.ListCustomerResponse{Total: 0, Items: nil}, nil
	}

	return &salesV1.ListCustomerResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *CustomerRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Customer.Query().
		Where(customer.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, salesV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *CustomerRepo) Get(ctx context.Context, req *salesV1.GetCustomerRequest) (*salesV1.Customer, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Customer.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *salesV1.GetCustomerRequest_Id:
		whereCond = append(whereCond, customer.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *CustomerRepo) Create(ctx context.Context, req *salesV1.CreateCustomerRequest) (*salesV1.Customer, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Customer.Create().
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
		r.log.Errorf("insert customer failed: %s", err.Error())
		return nil, salesV1.ErrorInternalServerError("insert customer failed")
	}

	return r.mapper.ToDTO(t), nil
}

func (r *CustomerRepo) Update(ctx context.Context, req *salesV1.UpdateCustomerRequest) (*salesV1.Customer, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &salesV1.CreateCustomerRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().Customer.UpdateOneID(req.GetId())
	builder.Where(customer.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(customer.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *salesV1.Customer) {
			builder.
				SetNillableCode(req.Data.Code).
				SetNillableName(req.Data.Name).
				SetNillableContact(req.Data.Contact).
				SetNillablePhone(req.Data.Phone).
				SetNillableCreditLimit(req.Data.CreditLimit).
				SetNillableCreditLimit(req.Data.CreditLimit).
				SetNillableEnable(req.Data.Enable).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(customer.FieldID, req.GetId()))
		},
	)

	return result, err
}

func (r *CustomerRepo) Delete(ctx context.Context, req *salesV1.DeleteCustomerRequest) error {
	if req == nil {
		return salesV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().Customer.Delete()
	delBuilder.Where(customer.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(customer.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return salesV1.ErrorNotFound("customer not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return salesV1.ErrorInternalServerError("delete failed")
	}

	return nil
}


// GetByCode 按编码取客户（信用额度校验用）；不存在返回 nil, nil。
func (r *CustomerRepo) GetByCode(ctx context.Context, code string) (*salesV1.Customer, error) {
	tid, _ := maybeTenantFromViewer(ctx)
	row, err := r.entClient.Client().Customer.Query().
		Where(customer.TenantIDEQ(tid), customer.CodeEQ(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query customer by code failed: %s", err.Error())
		return nil, salesV1.ErrorInternalServerError("query customer failed")
	}
	return r.mapper.ToDTO(row), nil
}
