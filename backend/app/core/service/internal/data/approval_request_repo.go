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
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/approvalrequest"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

type ApprovalRequestRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[approvalV1.ApprovalRequest, ent.ApprovalRequest]
	statusConverter *mapper.EnumTypeConverter[approvalV1.ApprovalRequest_Status, approvalrequest.Status]

	repository *entCrud.Repository[
		ent.ApprovalRequestQuery, ent.ApprovalRequestSelect,
		ent.ApprovalRequestCreate, ent.ApprovalRequestCreateBulk,
		ent.ApprovalRequestUpdate, ent.ApprovalRequestUpdateOne,
		ent.ApprovalRequestDelete,
		predicate.ApprovalRequest,
		approvalV1.ApprovalRequest, ent.ApprovalRequest,
	]
}

func NewApprovalRequestRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ApprovalRequestRepo {
	repo := &ApprovalRequestRepo{
		log:           ctx.NewLoggerHelper("approval_request/repo/core-service"),
		entClient:     entClient,
	}

	repo.init()

	return repo
}

func (r *ApprovalRequestRepo) init() {
	r.mapper = mapper.NewCopierMapper[approvalV1.ApprovalRequest, ent.ApprovalRequest]()
	r.statusConverter = mapper.NewEnumTypeConverter[approvalV1.ApprovalRequest_Status, approvalrequest.Status](approvalV1.ApprovalRequest_Status_name, approvalV1.ApprovalRequest_Status_value)

	r.repository = entCrud.NewRepository[
		ent.ApprovalRequestQuery, ent.ApprovalRequestSelect,
		ent.ApprovalRequestCreate, ent.ApprovalRequestCreateBulk,
		ent.ApprovalRequestUpdate, ent.ApprovalRequestUpdateOne,
		ent.ApprovalRequestDelete,
		predicate.ApprovalRequest,
		approvalV1.ApprovalRequest, ent.ApprovalRequest,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *ApprovalRequestRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().ApprovalRequest.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, approvalV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

func (r *ApprovalRequestRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalRequestResponse, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApprovalRequest.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &approvalV1.ListApprovalRequestResponse{Total: 0, Items: nil}, nil
	}

	return &approvalV1.ListApprovalRequestResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

func (r *ApprovalRequestRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().ApprovalRequest.Query().
		Where(approvalrequest.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, approvalV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

func (r *ApprovalRequestRepo) Get(ctx context.Context, req *approvalV1.GetApprovalRequestRequest) (*approvalV1.ApprovalRequest, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().ApprovalRequest.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *approvalV1.GetApprovalRequestRequest_Id:
		whereCond = append(whereCond, approvalrequest.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}

	return dto, err
}

func (r *ApprovalRequestRepo) Create(ctx context.Context, req *approvalV1.CreateApprovalRequestRequest) (*approvalV1.ApprovalRequest, error) {
	if req == nil || req.Data == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}

	// applicant 由服务端 viewer context 推导，忽略客户端传入值
	applicantID := req.Data.ApplicantId
	if uid, hasUser := viewerUserIDFromContext(ctx); hasUser {
		applicantID = trans.Ptr(uid)
	}

	builder := r.entClient.Client().ApprovalRequest.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetNillableTitle(req.Data.Title).
		SetNillableBizType(req.Data.BizType).
		SetNillableBizRef(req.Data.BizRef).
		SetNillableSummary(req.Data.Summary).
		// 状态强制为 PENDING，忽略客户端传入值
		SetStatus(approvalrequest.StatusPending).
		SetNillableApplicantID(applicantID).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert approval_request failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("insert approval_request failed")
	}

	return r.mapper.ToDTO(t), nil
}

// TransitionStatus 原子状态迁移：仅当当前状态为 from 时才更新为 to，
// 同时盖章审批人与审批意见。0 行受影响说明已被并发变更，返回 Conflict。
func (r *ApprovalRequestRepo) TransitionStatus(
	ctx context.Context,
	id uint32,
	from, to approvalV1.ApprovalRequest_Status,
	comment *string,
) (*approvalV1.ApprovalRequest, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().ApprovalRequest.Update().
		Where(approvalrequest.IDEQ(id)).
		Where(approvalrequest.StatusEQ(*r.statusConverter.ToEntity(trans.Ptr(from)))).
		SetStatus(*r.statusConverter.ToEntity(trans.Ptr(to))).
		SetUpdatedAt(time.Now())

	if hasTenant {
		builder.Where(approvalrequest.TenantIDEQ(tid))
	}
	if hasUser {
		// 审批人由服务端 viewer context 推导
		builder.SetApproverID(callerUserID)
		builder.SetUpdatedBy(callerUserID)
	}
	if comment != nil {
		builder.SetNillableComment(comment)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("transition approval status failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("transition approval status failed")
	}
	if n == 0 {
		return nil, approvalV1.ErrorConflict("approval request status changed concurrently")
	}

	return r.Get(ctx, &approvalV1.GetApprovalRequestRequest{
		QueryBy: &approvalV1.GetApprovalRequestRequest_Id{Id: id},
	})
}

// CancelAsApplicant 撤销：仅当 PENDING 且申请人为 caller 时更新为 CANCELLED。
func (r *ApprovalRequestRepo) CancelAsApplicant(ctx context.Context, id uint32, applicantID uint32) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().ApprovalRequest.Update().
		Where(approvalrequest.IDEQ(id)).
		Where(approvalrequest.StatusEQ(approvalrequest.StatusPending)).
		Where(approvalrequest.ApplicantIDEQ(applicantID)).
		SetStatus(approvalrequest.StatusCancelled).
		SetUpdatedBy(applicantID).
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(approvalrequest.TenantIDEQ(tid))
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("cancel approval failed: %s", err.Error())
		return approvalV1.ErrorInternalServerError("cancel approval failed")
	}
	if n == 0 {
		return approvalV1.ErrorConflict("approval request not pending or not owned by caller")
	}

	return nil
}

func (r *ApprovalRequestRepo) Delete(ctx context.Context, req *approvalV1.DeleteApprovalRequestRequest) error {
	if req == nil {
		return approvalV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().ApprovalRequest.Delete()
	delBuilder.Where(approvalrequest.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(approvalrequest.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return approvalV1.ErrorNotFound("approval_request not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return approvalV1.ErrorInternalServerError("delete failed")
	}

	return nil
}
