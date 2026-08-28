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
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/receipt"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

type ReceiptRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[financeV1.Receipt, ent.Receipt]
	methodConverter *mapper.EnumTypeConverter[financeV1.Receipt_Method, receipt.Method]
	statusConverter *mapper.EnumTypeConverter[financeV1.Receipt_Status, receipt.Status]

	repository *entCrud.Repository[
		ent.ReceiptQuery, ent.ReceiptSelect,
		ent.ReceiptCreate, ent.ReceiptCreateBulk,
		ent.ReceiptUpdate, ent.ReceiptUpdateOne,
		ent.ReceiptDelete,
		predicate.Receipt,
		financeV1.Receipt, ent.Receipt,
	]
}

func NewReceiptRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ReceiptRepo {
	repo := &ReceiptRepo{
		log:       ctx.NewLoggerHelper("receipt/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *ReceiptRepo) init() {
	r.mapper = mapper.NewCopierMapper[financeV1.Receipt, ent.Receipt]()
	r.methodConverter = mapper.NewEnumTypeConverter[financeV1.Receipt_Method, receipt.Method](financeV1.Receipt_Method_name, financeV1.Receipt_Method_value)
	r.statusConverter = mapper.NewEnumTypeConverter[financeV1.Receipt_Status, receipt.Status](financeV1.Receipt_Status_name, financeV1.Receipt_Status_value)

	r.repository = entCrud.NewRepository[
		ent.ReceiptQuery, ent.ReceiptSelect,
		ent.ReceiptCreate, ent.ReceiptCreateBulk,
		ent.ReceiptUpdate, ent.ReceiptUpdateOne,
		ent.ReceiptDelete,
		predicate.Receipt,
		financeV1.Receipt, ent.Receipt,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.methodConverter.NewConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *ReceiptRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().Receipt.Query()
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

func (r *ReceiptRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceiptResponse, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Receipt.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &financeV1.ListReceiptResponse{Total: 0, Items: nil}, nil
	}

	return &financeV1.ListReceiptResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ReceiptRepo) Get(ctx context.Context, req *financeV1.GetReceiptRequest) (*financeV1.Receipt, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().Receipt.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *financeV1.GetReceiptRequest_Id:
		whereCond = append(whereCond, receipt.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

// TransitionStatus 原子状态迁移（PENDING→APPLIED/REJECTED）。
func (r *ReceiptRepo) TransitionStatus(
	ctx context.Context,
	id uint32,
	from, to financeV1.Receipt_Status,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Receipt.Update().
		Where(receipt.IDEQ(id)).
		Where(receipt.StatusEQ(*r.statusConverter.ToEntity(trans.Ptr(from)))).
		SetStatus(*r.statusConverter.ToEntity(trans.Ptr(to)))
	if hasTenant {
		builder.Where(receipt.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("transition receipt status failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("transition receipt status failed")
	}
	if n == 0 {
		return financeV1.ErrorConflict("receipt status changed concurrently")
	}
	return nil
}

// Create 落收款流水（应收单的 paid_amount/status 由 ApplyReceipt 负责）。
func (r *ReceiptRepo) Create(ctx context.Context, req *financeV1.CreateReceiptRequest) (*financeV1.Receipt, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().Receipt.Create().
		SetReceiptNumber("RT"+fmt.Sprintf("%d", time.Now().UnixMilli())).
		SetNillableReceivableID(req.Data.ReceivableId).
		SetNillableAmount(req.Data.Amount).
		SetNillableMethod(r.methodConverter.ToEntity(req.Data.Method)).
		SetStatus(receipt.StatusPending).
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
		r.log.Errorf("insert receipt failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("insert receipt failed")
	}

	return r.mapper.ToDTO(t), nil
}
