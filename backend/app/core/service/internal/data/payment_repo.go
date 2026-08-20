package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/payment"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

type PaymentRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[financeV1.Payment, ent.Payment]
	methodConverter *mapper.EnumTypeConverter[financeV1.Payment_Method, payment.Method]

	repository *entCrud.Repository[
		ent.PaymentQuery, ent.PaymentSelect,
		ent.PaymentCreate, ent.PaymentCreateBulk,
		ent.PaymentUpdate, ent.PaymentUpdateOne,
		ent.PaymentDelete,
		predicate.Payment,
		financeV1.Payment, ent.Payment,
	]
}

func NewPaymentRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PaymentRepo {
	repo := &PaymentRepo{
		log:       ctx.NewLoggerHelper("payment/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *PaymentRepo) init() {
	r.mapper = mapper.NewCopierMapper[financeV1.Payment, ent.Payment]()
	r.methodConverter = mapper.NewEnumTypeConverter[financeV1.Payment_Method, payment.Method](financeV1.Payment_Method_name, financeV1.Payment_Method_value)

	r.repository = entCrud.NewRepository[
		ent.PaymentQuery, ent.PaymentSelect,
		ent.PaymentCreate, ent.PaymentCreateBulk,
		ent.PaymentUpdate, ent.PaymentUpdateOne,
		ent.PaymentDelete,
		predicate.Payment,
		financeV1.Payment, ent.Payment,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())
}

func (r *PaymentRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Payment.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, financeV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *PaymentRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPaymentResponse, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Payment.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &financeV1.ListPaymentResponse{Total: 0, Items: nil}, nil
	}

	return &financeV1.ListPaymentResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PaymentRepo) Get(ctx context.Context, req *financeV1.GetPaymentRequest) (*financeV1.Payment, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Payment.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *financeV1.GetPaymentRequest_Id:
		whereCond = append(whereCond, payment.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// Create 落付款流水（应付单的 paid_amount/status 由 ApplyPayment 负责）。
func (r *PaymentRepo) Create(ctx context.Context, req *financeV1.CreatePaymentRequest) (*financeV1.Payment, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Payment.Create().
		SetPaymentNumber("PM"+fmt.Sprintf("%d", time.Now().UnixMilli())).
		SetNillablePayableID(req.Data.PayableId).
		SetNillableAmount(req.Data.Amount).
		SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
		SetNillableRemark(req.Data.Remark).
		SetCreatedAt(time.Now())
	if hasTenant {
		builder.SetNillableTenantID(trans.Ptr(tid))
	} else {
		builder.SetNillableTenantID(req.Data.TenantId)
	}
	if hasUser {
		// 操作人由服务端从登录态推导
		builder.SetCreatedBy(callerUserID)
	}

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert payment failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("insert payment failed")
	}

	return r.mapper.ToDTO(t), nil
}
