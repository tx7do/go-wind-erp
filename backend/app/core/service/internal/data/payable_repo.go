package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/payable"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

type PayableRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[financeV1.Payable, ent.Payable]
	statusConverter *mapper.EnumTypeConverter[financeV1.Payable_Status, payable.Status]

	repository *entCrud.Repository[
		ent.PayableQuery, ent.PayableSelect,
		ent.PayableCreate, ent.PayableCreateBulk,
		ent.PayableUpdate, ent.PayableUpdateOne,
		ent.PayableDelete,
		predicate.Payable,
		financeV1.Payable, ent.Payable,
	]
}

func NewPayableRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PayableRepo {
	repo := &PayableRepo{
		log:       ctx.NewLoggerHelper("payable/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *PayableRepo) init() {
	r.mapper = mapper.NewCopierMapper[financeV1.Payable, ent.Payable]()
	r.statusConverter = mapper.NewEnumTypeConverter[financeV1.Payable_Status, payable.Status](financeV1.Payable_Status_name, financeV1.Payable_Status_value)

	r.repository = entCrud.NewRepository[
		ent.PayableQuery, ent.PayableSelect,
		ent.PayableCreate, ent.PayableCreateBulk,
		ent.PayableUpdate, ent.PayableUpdateOne,
		ent.PayableDelete,
		predicate.Payable,
		financeV1.Payable, ent.Payable,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PayableRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Payable.Query()
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

func (r *PayableRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPayableResponse, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Payable.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &financeV1.ListPayableResponse{Total: 0, Items: nil}, nil
	}

	return &financeV1.ListPayableResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *PayableRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Payable.Query().
		Where(payable.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, financeV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *PayableRepo) Get(ctx context.Context, req *financeV1.GetPayableRequest) (*financeV1.Payable, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Payable.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *financeV1.GetPayableRequest_Id:
		whereCond = append(whereCond, payable.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// Create 建应付单（状态强制 PENDING、已付强制 0；采购联动/手工建账共用）。
func (r *PayableRepo) Create(ctx context.Context, req *financeV1.CreatePayableRequest) (*financeV1.Payable, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Payable.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetPayableNumber("AP"+fmt.Sprintf("%d", time.Now().UnixMilli())).
		SetNillablePoRef(req.Data.PoRef).
		SetNillableSupplierCode(req.Data.SupplierCode).
		SetNillableAmount(req.Data.Amount).
		SetNillableDueDate(protoTime(req.Data.DueDate)).
		SetPaidAmount(0).
		SetStatus(payable.StatusPending).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert payable failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("insert payable failed")
	}

	return r.mapper.ToDTO(t), nil
}

// CancelAsUnpaid 取消：仅 PENDING 且 paid_amount=0（条件更新保证并发安全）。
func (r *PayableRepo) CancelAsUnpaid(ctx context.Context, id uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Payable.Update().
		Where(payable.IDEQ(id)).
		Where(payable.StatusEQ(payable.StatusPending)).
		Where(payable.PaidAmountEQ(0)).
		SetStatus(payable.StatusCancelled).
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(payable.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("cancel payable failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("cancel payable failed")
	}
	if n == 0 {
		return financeV1.ErrorConflict("payable is not pending-unpaid")
	}
	return nil
}

// DeleteAsUnpaid 删除：仅 PENDING 且未付款（终态审计与在途账不可抹除）。
func (r *PayableRepo) DeleteAsUnpaid(ctx context.Context, id uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().Payable.Delete().
		Where(payable.IDEQ(id)).
		Where(payable.StatusEQ(payable.StatusPending)).
		Where(payable.PaidAmount(0))
	if hasTenant {
		builder.Where(payable.TenantIDEQ(tid))
	}

	n, err := builder.Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return financeV1.ErrorNotFound("payable not found")
		}
		r.log.Errorf("delete payable failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("delete failed")
	}
	if n == 0 {
		return financeV1.ErrorConflict("payable is not pending-unpaid")
	}
	return nil
}

// ApplyPayment 原子付款入账：单条 UPDATE 同时累加已付并按新值推导状态，
// WHERE 子句在当前值上校验不超付——并发付款竞态被封闭在 SQL 层。
// 返回更新后的应付单。0 行受影响 → 超付或不可付款状态。
func (r *PayableRepo) ApplyPayment(ctx context.Context, payableID uint32, amount int64) (*financeV1.Payable, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Payable.Update().
		Where(payable.IDEQ(payableID)).
		// 仅未结清/未取消的应付可继续付款
		Where(payable.StatusIn(payable.StatusPending, payable.StatusPartial)).
		// 防超付：当前已付 + 本次 ≤ 总额（对当前值求值，非读到的旧值）
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d <= %s",
				s.C(payable.FieldPaidAmount), amount, s.C(payable.FieldAmount),
			)))
		}).
		AddPaidAmount(amount).
		// 状态按累加后的终值推导（CASE 在 DB 内对累加结果求值）
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(payable.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}
	builder.Modify(func(u *sql.UpdateBuilder) {
		u.Set(
			payable.FieldStatus,
			sql.Expr(fmt.Sprintf(
				"CASE WHEN %s + %d >= %s THEN '%s' ELSE '%s' END",
				payable.FieldPaidAmount, amount, payable.FieldAmount,
				payable.StatusSettled, payable.StatusPartial,
			)),
		)
	})

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("apply payment failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("apply payment failed")
	}
	if n == 0 {
		return nil, financeV1.ErrorConflict("payment exceeds payable amount or payable not payable")
	}

	return r.Get(ctx, &financeV1.GetPayableRequest{
		QueryBy: &financeV1.GetPayableRequest_Id{Id: payableID},
	})
}


// protoTime *timestamppb.Timestamp → *time.Time（nil 安全）。
func protoTime(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	tt := t.AsTime()
	return &tt
}
