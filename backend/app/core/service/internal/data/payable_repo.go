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

// AgingReport 按 due_date 对未结清（PENDING/PARTIAL）应付做分桶聚合：
// overdue / 0_7 / 8_30 / 31_90 / over_90 / no_due_date。
//
// ent 的 GroupBy 只接受真实列（无法 GROUP BY 计算列 CASE），故 SQL 端
// 只按 due_date 聚合金额，分桶在客户端按距今天数完成。NULL due_date
// （手工建账或无账期）入 no_due_date 桶。
func (r *PayableRepo) AgingReport(ctx context.Context) ([]*financeV1.AgingBucket, error) {
	// 每行汇总未清余额（amount − paid_amount）而非面值 amount，否则对
	// PARTIAL（部分付款）行会高估未清余额。COUNT(*) 仅作该桶行数，由
	// ent.As 别名至下方 scan 结构字段；详见 ent.AggregateFunc 文档。
	type agingRow struct {
		DueDate *time.Time `sql:"due_date"`
		Total   int64      `sql:"total"`
		Count   int64      `sql:"count"`
	}
	var rows []agingRow

	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().Payable.Query().
		Where(payable.StatusIn(payable.StatusPending, payable.StatusPartial))
	if hasTenant {
		builder.Where(payable.TenantIDEQ(tid))
	}
	if err := builder.GroupBy(payable.FieldDueDate).
		Aggregate(
			ent.As(outstandingBalanceSum, "total"),
			ent.As(ent.Count(), "count"),
		).
		Scan(ctx, &rows); err != nil {
		r.log.Errorf("aging report query failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("aging report query failed")
	}

	labels := []string{"overdue", "0_7", "8_30", "31_90", "over_90", "no_due_date"}
	agg := map[string]*financeV1.AgingBucket{}
	for _, l := range labels {
		lbl := l
		zero := int64(0)
		// Count/TotalAmount 初始化为 0 的指针，使累加路径恒有非 nil 接收方。
		agg[l] = &financeV1.AgingBucket{Bucket: &lbl, Count: &zero, TotalAmount: &zero}
	}
	now := time.Now()
	for _, row := range rows {
		var label string
		if row.DueDate == nil {
			label = "no_due_date"
		} else {
			days := int(row.DueDate.Sub(now).Hours() / 24)
			switch {
			case days < 0:
				label = "overdue"
			case days <= 7:
				label = "0_7"
			case days <= 30:
				label = "8_30"
			case days <= 90:
				label = "31_90"
			default:
				label = "over_90"
			}
		}
		// 累加该桶的未清余额与行数（旧实现把 Count 覆写成 1，计数恒无效）。
		c := *agg[label].Count + row.Count
		agg[label].Count = &c
		t := *agg[label].TotalAmount + row.Total
		agg[label].TotalAmount = &t
	}
	out := make([]*financeV1.AgingBucket, 0, len(labels))
	for _, l := range labels {
		out = append(out, agg[l])
	}
	return out, nil
}

// outstandingBalanceSum 按 due_date 分组聚合未清余额之和
// SUM(COALESCE(amount,0) − COALESCE(paid_amount,0))。直接以 s.C 拼接带
// 引号列名并外包 SUM(...)，绕过 sql.Sum 对普通标识符的强制 Quote——差值
// 表达式不是单一标识符，经 Quote 会损坏 SQL。ent.As 将其别名至 "total"
// 以匹配 scan 结构标签。COALESCE 兜底空值：两列虽 Default(0)，但 schema
// 仍为 Optional+Nillable。ent.Count() 产出 COUNT(*)，同样经 As 别名 "count"。
func outstandingBalanceSum(s *sql.Selector) string {
	expr := fmt.Sprintf(
		"COALESCE(%s,0) - COALESCE(%s,0)",
		s.C(payable.FieldAmount), s.C(payable.FieldPaidAmount),
	)
	return fmt.Sprintf("SUM(%s)", expr)
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
