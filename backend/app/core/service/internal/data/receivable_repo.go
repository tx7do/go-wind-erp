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

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/receivable"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

type ReceivableRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[financeV1.Receivable, ent.Receivable]
	statusConverter *mapper.EnumTypeConverter[financeV1.Receivable_Status, receivable.Status]

	repository *entCrud.Repository[
		ent.ReceivableQuery, ent.ReceivableSelect,
		ent.ReceivableCreate, ent.ReceivableCreateBulk,
		ent.ReceivableUpdate, ent.ReceivableUpdateOne,
		ent.ReceivableDelete,
		predicate.Receivable,
		financeV1.Receivable, ent.Receivable,
	]
}

func NewReceivableRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ReceivableRepo {
	repo := &ReceivableRepo{
		log:       ctx.NewLoggerHelper("receivable/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *ReceivableRepo) init() {
	r.mapper = mapper.NewCopierMapper[financeV1.Receivable, ent.Receivable]()
	r.statusConverter = mapper.NewEnumTypeConverter[financeV1.Receivable_Status, receivable.Status](financeV1.Receivable_Status_name, financeV1.Receivable_Status_value)

	r.repository = entCrud.NewRepository[
		ent.ReceivableQuery, ent.ReceivableSelect,
		ent.ReceivableCreate, ent.ReceivableCreateBulk,
		ent.ReceivableUpdate, ent.ReceivableUpdateOne,
		ent.ReceivableDelete,
		predicate.Receivable,
		financeV1.Receivable, ent.Receivable,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *ReceivableRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Receivable.Query()
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

func (r *ReceivableRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceivableResponse, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Receivable.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &financeV1.ListReceivableResponse{Total: 0, Items: nil}, nil
	}

	return &financeV1.ListReceivableResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ReceivableRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().Receivable.Query().
		Where(receivable.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, financeV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ReceivableRepo) Get(ctx context.Context, req *financeV1.GetReceivableRequest) (*financeV1.Receivable, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Receivable.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *financeV1.GetReceivableRequest_Id:
		whereCond = append(whereCond, receivable.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// Create 建应收单（状态强制 PENDING、已收强制 0；销售联动/手工建账共用）。
func (r *ReceivableRepo) Create(ctx context.Context, req *financeV1.CreateReceivableRequest) (*financeV1.Receivable, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Receivable.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetReceivableNumber("AR"+fmt.Sprintf("%d", time.Now().UnixMilli())).
		SetNillableSoRef(req.Data.SoRef).
		SetNillableCustomerCode(req.Data.CustomerCode).
		SetNillableAmount(req.Data.Amount).
		SetNillableDueDate(protoTime(req.Data.DueDate)).
		SetPaidAmount(0).
		SetStatus(receivable.StatusPending).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert receivable failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("insert receivable failed")
	}

	return r.mapper.ToDTO(t), nil
}

// AgingReport 按 due_date 对未结清（PENDING/PARTIAL）应收做分桶聚合：
// overdue / 0_7 / 8_30 / 31_90 / over_90 / no_due_date。
//
// ent 的 GroupBy 只接受真实列（无法 GROUP BY 计算列 CASE），故 SQL 端
// 只按 due_date 聚合金额，分桶在客户端按距今天数完成。NULL due_date
// （手工建账或无账期）入 no_due_date 桶。
func (r *ReceivableRepo) AgingReport(ctx context.Context) ([]*financeV1.AgingBucket, error) {
	// 每行汇总未清余额（amount − paid_amount）而非面值 amount，否则对
	// PARTIAL（部分收款）行会高估未清余额。COUNT(*) 仅作该桶行数，由
	// ent.As 别名至下方 scan 结构字段；详见 ent.AggregateFunc 文档。
	type agingRow struct {
		DueDate *time.Time `sql:"due_date"`
		Total   int64      `sql:"total"`
		Count   int64      `sql:"count"`
	}
	var rows []agingRow

	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().Receivable.Query().
		Where(receivable.StatusIn(receivable.StatusPending, receivable.StatusPartial))
	if hasTenant {
		builder.Where(receivable.TenantIDEQ(tid))
	}
	if err := builder.GroupBy(receivable.FieldDueDate).
		Aggregate(
			ent.As(outstandingReceivableBalanceSum, "total"),
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

// outstandingReceivableBalanceSum 按 due_date 分组聚合未清余额之和
// SUM(COALESCE(amount,0) − COALESCE(paid_amount,0))。直接以 s.C 拼接带
// 引号列名并外包 SUM(...)，绕过 sql.Sum 对普通标识符的强制 Quote——差值
// 表达式不是单一标识符，经 Quote 会损坏 SQL。ent.As 将其别名至 "total"
// 以匹配 scan 结构标签。COALESCE 兜底空值：两列虽 Default(0)，但 schema
// 仍为 Optional+Nillable。ent.Count() 产出 COUNT(*)，同样经 As 别名 "count"。
func outstandingReceivableBalanceSum(s *sql.Selector) string {
	expr := fmt.Sprintf(
		"COALESCE(%s,0) - COALESCE(%s,0)",
		s.C(receivable.FieldAmount), s.C(receivable.FieldPaidAmount),
	)
	return fmt.Sprintf("SUM(%s)", expr)
}

// CancelAsUnpaid 取消：仅 PENDING 且 paid_amount=0（条件更新保证并发安全）。
func (r *ReceivableRepo) CancelAsUnpaid(ctx context.Context, id uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Receivable.Update().
		Where(receivable.IDEQ(id)).
		Where(receivable.StatusEQ(receivable.StatusPending)).
		Where(receivable.PaidAmountEQ(0)).
		SetStatus(receivable.StatusCancelled).
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(receivable.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("cancel receivable failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("cancel receivable failed")
	}
	if n == 0 {
		return financeV1.ErrorConflict("receivable is not pending-unpaid")
	}
	return nil
}

// DeleteAsUnpaid 删除：仅 PENDING 且未收款（终态审计与在途账不可抹除）。
func (r *ReceivableRepo) DeleteAsUnpaid(ctx context.Context, id uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().Receivable.Delete().
		Where(receivable.IDEQ(id)).
		Where(receivable.StatusEQ(receivable.StatusPending)).
		Where(receivable.PaidAmount(0))
	if hasTenant {
		builder.Where(receivable.TenantIDEQ(tid))
	}

	n, err := builder.Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return financeV1.ErrorNotFound("receivable not found")
		}
		r.log.Errorf("delete receivable failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("delete failed")
	}
	if n == 0 {
		return financeV1.ErrorConflict("receivable is not pending-unpaid")
	}
	return nil
}

// ApplyReceipt 原子收款入账：单条 UPDATE 同时累加已收并按新值推导状态，
// WHERE 子句在当前值上校验不超收——并发收款竞态被封闭在 SQL 层。
// 返回更新后的应收单。0 行受影响 → 超收或不可收款状态。
func (r *ReceivableRepo) ApplyReceipt(ctx context.Context, receivableID uint32, amount int64) (*financeV1.Receivable, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Receivable.Update().
		Where(receivable.IDEQ(receivableID)).
		// 仅未结清/未取消的应收可继续收款
		Where(receivable.StatusIn(receivable.StatusPending, receivable.StatusPartial)).
		// 防超收：当前已收 + 本次 ≤ 总额（对当前值求值，非读到的旧值）
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d <= %s",
				s.C(receivable.FieldPaidAmount), amount, s.C(receivable.FieldAmount),
			)))
		}).
		AddPaidAmount(amount).
		// 状态按累加后的终值推导（CASE 在 DB 内对累加结果求值）
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(receivable.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}
	builder.Modify(func(u *sql.UpdateBuilder) {
		u.Set(
			receivable.FieldStatus,
			sql.Expr(fmt.Sprintf(
				"CASE WHEN %s + %d >= %s THEN '%s' ELSE '%s' END",
				receivable.FieldPaidAmount, amount, receivable.FieldAmount,
				receivable.StatusSettled, receivable.StatusPartial,
			)),
		)
	})

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("apply receipt failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("apply receipt failed")
	}
	if n == 0 {
		return nil, financeV1.ErrorConflict("receipt exceeds receivable amount or receivable not receivable")
	}

	return r.Get(ctx, &financeV1.GetReceivableRequest{
		QueryBy: &financeV1.GetReceivableRequest_Id{Id: receivableID},
	})
}
