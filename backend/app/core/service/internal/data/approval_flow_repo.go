package data

import (
	"context"
	"time"

	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/trans"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/approvalflow"
	"go-wind-erp/app/core/service/internal/data/ent/approvalflowstep"
	"go-wind-erp/app/core/service/internal/data/ent/role"
	"go-wind-erp/app/core/service/internal/data/ent/userrole"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"

	appViewer "go-wind-erp/pkg/entgo/viewer"
)

// ApprovalFlowRepo 审批流模板仓储（多级审批）。
//
// 流程行（apr_approval_flows）+ 级定义（apr_approval_flow_steps）。级随
// 流程整体替换（编辑=删全级重建），读取时按 seq 升序拼装。快照语义：
// 审批请求创建时只取级数与流程ID，在途单不受后续流程编辑影响。
type ApprovalFlowRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewApprovalFlowRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *ApprovalFlowRepo {
	return &ApprovalFlowRepo{
		log:       ctx.NewLoggerHelper("approval_flow/repo/core-service"),
		entClient: entClient,
	}
}

func (r *ApprovalFlowRepo) toFlowDTO(f *ent.ApprovalFlow, steps []*ent.ApprovalFlowStep) *approvalV1.ApprovalFlow {
	dto := &approvalV1.ApprovalFlow{
		Id:       &f.ID,
		BizType:  f.BizType,
		Name:     f.Name,
		TenantId: f.TenantID,
	}
	if f.Status != nil {
		dto.Status = trans.Ptr(string(*f.Status))
	}
	for _, st := range steps {
		step := &approvalV1.ApprovalFlowStep{
			Id:       &st.ID,
			Name:     st.Name,
			RoleCode: st.RoleCode,
		}
		if st.FlowID != nil {
			fid := st.FlowID
			step.FlowId = fid
		}
		if st.Seq != nil {
			seq := st.Seq
			step.Seq = seq
		}
		dto.Steps = append(dto.Steps, step)
	}
	return dto
}

// FindActiveSteps 取生效流程的级定义（按 seq 升序）；无流程返回 nil。
func (r *ApprovalFlowRepo) FindActiveSteps(
	ctx context.Context,
	bizType string,
) (*approvalV1.ApprovalFlow, error) {
	tid, _ := maybeTenantFromViewer(ctx)

	flow, err := r.entClient.Client().ApprovalFlow.Query().
		Where(
			approvalflow.TenantIDEQ(tid),
			approvalflow.BizTypeEQ(bizType),
			approvalflow.StatusEQ(approvalflow.StatusOn),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("query approval flow failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("query approval flow failed")
	}

	steps, err := r.entClient.Client().ApprovalFlowStep.Query().
		Where(approvalflowstep.FlowIDEQ(flow.ID)).
		Order(ent.Asc(approvalflowstep.FieldSeq)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query approval flow steps failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("query approval flow steps failed")
	}
	return r.toFlowDTO(flow, steps), nil
}

// GetStepRole 取流程某级的审批角色编码；越级或流程缺失返回空串。
func (r *ApprovalFlowRepo) GetStepRole(
	ctx context.Context,
	flowID uint32,
	step uint32,
) (string, error) {
	st, err := r.entClient.Client().ApprovalFlowStep.Query().
		Where(
			approvalflowstep.FlowIDEQ(flowID),
			approvalflowstep.SeqEQ(step),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		r.log.Errorf("query approval flow step failed: %s", err.Error())
		return "", approvalV1.ErrorInternalServerError("query approval flow step failed")
	}
	if st.RoleCode == nil {
		return "", nil
	}
	return *st.RoleCode, nil
}

// List 流程列表（含级定义，按创建倒序 + Go 侧分页，流程行数有限）。
func (r *ApprovalFlowRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalFlowResponse, error) {
	if req == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	q := r.entClient.Client().ApprovalFlow.Query()
	if hasTenant {
		q = q.Where(approvalflow.TenantIDEQ(tid))
	}
	flows, err := q.Order(ent.Desc(approvalflow.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list approval flows failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("list approval flows failed")
	}

	flowIDs := make([]uint32, 0, len(flows))
	for _, f := range flows {
		flowIDs = append(flowIDs, f.ID)
	}
	stepRows, err := r.entClient.Client().ApprovalFlowStep.Query().
		Where(approvalflowstep.FlowIDIn(flowIDs...)).
		Order(ent.Asc(approvalflowstep.FieldSeq)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list approval flow steps failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("list approval flow steps failed")
	}
	stepsByFlow := map[uint32][]*ent.ApprovalFlowStep{}
	for _, st := range stepRows {
		if st.FlowID == nil {
			continue
		}
		stepsByFlow[*st.FlowID] = append(stepsByFlow[*st.FlowID], st)
	}

	items := make([]*approvalV1.ApprovalFlow, 0, len(flows))
	for _, f := range flows {
		items = append(items, r.toFlowDTO(f, stepsByFlow[f.ID]))
	}
	return &approvalV1.ListApprovalFlowResponse{Total: uint64(len(items)), Items: items}, nil
}

// Get 流程详情（含级定义）。
func (r *ApprovalFlowRepo) Get(ctx context.Context, id uint32) (*approvalV1.ApprovalFlow, error) {
	flow, err := r.entClient.Client().ApprovalFlow.Query().
		Where(approvalflow.IDEQ(id)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, approvalV1.ErrorNotFound("approval flow not found")
		}
		r.log.Errorf("get approval flow failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("get approval flow failed")
	}
	steps, err := r.entClient.Client().ApprovalFlowStep.Query().
		Where(approvalflowstep.FlowIDEQ(id)).
		Order(ent.Asc(approvalflowstep.FieldSeq)).
		All(ctx)
	if err != nil {
		return nil, approvalV1.ErrorInternalServerError("get approval flow steps failed")
	}
	return r.toFlowDTO(flow, steps), nil
}

// ExistsActiveByBizType 同租户同业务类型是否已有（其它）生效流程。
func (r *ApprovalFlowRepo) ExistsActiveByBizType(
	ctx context.Context,
	bizType string,
	excludeFlowID uint32,
) (bool, error) {
	tid, _ := maybeTenantFromViewer(ctx)
	q := r.entClient.Client().ApprovalFlow.Query().
		Where(
			approvalflow.TenantIDEQ(tid),
			approvalflow.BizTypeEQ(bizType),
		)
	if excludeFlowID != 0 {
		q = q.Where(approvalflow.IDNEQ(excludeFlowID))
	}
	return q.Exist(ctx)
}

// Create 创建流程 + 级定义（单事务）。
func (r *ApprovalFlowRepo) Create(ctx context.Context, data *approvalV1.ApprovalFlow) error {
	tid, _ := maybeTenantFromViewer(ctx)

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return approvalV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	flow, err := tx.ApprovalFlow.Create().
		SetTenantID(tid).
		SetNillableBizType(data.BizType).
		SetNillableName(data.Name).
		SetStatus(approvalflow.StatusOn).
		SetNillableRemark(data.Remark).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert approval flow failed: %s", err.Error())
		return approvalV1.ErrorInternalServerError("insert approval flow failed")
	}
	if err = r.replaceStepsTx(ctx, tx, tid, flow.ID, data.GetSteps()); err != nil {
		return err
	}
	return nil
}

// Update 更新流程 + 级整体替换（单事务）。
func (r *ApprovalFlowRepo) Update(ctx context.Context, data *approvalV1.ApprovalFlow) error {
	tid, _ := maybeTenantFromViewer(ctx)

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return approvalV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	builder := tx.ApprovalFlow.UpdateOneID(data.GetId()).
		SetNillableBizType(data.BizType).
		SetNillableName(data.Name).
		SetNillableRemark(data.Remark).
		SetUpdatedAt(time.Now())
	if data.Status != nil {
		builder.SetStatus(*r.flowStatusToEntity(*data.Status))
	}
	if _, err = builder.Save(ctx); err != nil {
		r.log.Errorf("update approval flow failed: %s", err.Error())
		return approvalV1.ErrorInternalServerError("update approval flow failed")
	}

	// 级整体替换：删旧插新。
	if _, err = tx.ApprovalFlowStep.Delete().
		Where(approvalflowstep.FlowIDEQ(data.GetId())).
		Exec(ctx); err != nil {
		r.log.Errorf("replace approval flow steps failed: %s", err.Error())
		return approvalV1.ErrorInternalServerError("replace approval flow steps failed")
	}
	if err = r.replaceStepsTx(ctx, tx, tid, data.GetId(), data.GetSteps()); err != nil {
		return err
	}
	return nil
}

func (r *ApprovalFlowRepo) flowStatusToEntity(s string) *approvalflow.Status {
	switch s {
	case "OFF":
		v := approvalflow.StatusOff
		return &v
	default:
		v := approvalflow.StatusOn
		return &v
	}
}

func (r *ApprovalFlowRepo) replaceStepsTx(
	ctx context.Context,
	tx *ent.Tx,
	tenantID uint32,
	flowID uint32,
	steps []*approvalV1.ApprovalFlowStep,
) error {
	if len(steps) == 0 {
		return approvalV1.ErrorBadRequest("approval flow requires at least one step")
	}
	for i, st := range steps {
		if st == nil || st.GetRoleCode() == "" {
			return approvalV1.ErrorBadRequest("approval flow step requires role_code")
		}
		name := st.GetName()
		if name == "" {
			name = "第 " + strconv.Itoa(i+1) + " 级"
		}
		if _, err := tx.ApprovalFlowStep.Create().
			SetTenantID(tenantID).
			SetFlowID(flowID).
			SetSeq(uint32(i + 1)).
			SetName(name).
			SetRoleCode(st.GetRoleCode()).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			r.log.Errorf("insert approval flow step failed: %s", err.Error())
			return approvalV1.ErrorInternalServerError("insert approval flow step failed")
		}
	}
	return nil
}

// UserIDsByRole 取持有指定角色编码的全部用户 ID（下一级审批通知用）。
func (r *ApprovalFlowRepo) UserIDsByRole(ctx context.Context, roleCode string) ([]uint32, error) {
	// 同 UserHoldsRole：角色目录走系统视图。
	roleIDs, err := r.entClient.Client().Role.Query().
		Where(role.CodeEQ(roleCode)).
		IDs(appViewer.NewSystemViewerContext(ctx))
	if err != nil || len(roleIDs) == 0 {
		return nil, nil
	}
	ids, err := r.entClient.Client().UserRole.Query().
		Where(userrole.RoleIDIn(roleIDs...)).
		IDs(ctx)
	if err != nil {
		r.log.Errorf("query user ids by role failed: %s", err.Error())
		return nil, approvalV1.ErrorInternalServerError("query user ids by role failed")
	}
	return ids, nil
}

// Delete 删除流程 + 级（硬删，连带）。
func (r *ApprovalFlowRepo) Delete(ctx context.Context, id uint32) error {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return approvalV1.ErrorInternalServerError("start transaction failed")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	if _, err = tx.ApprovalFlowStep.Delete().
		Where(approvalflowstep.FlowIDEQ(id)).
		Exec(ctx); err != nil {
		return approvalV1.ErrorInternalServerError("delete approval flow steps failed")
	}
	if _, err = tx.ApprovalFlow.Delete().Where(approvalflow.IDEQ(id)).Exec(ctx); err != nil {
		r.log.Errorf("delete approval flow failed: %s", err.Error())
		return approvalV1.ErrorInternalServerError("delete approval flow failed")
	}
	return nil
}

// UserHoldsRole 用户是否持有指定角色编码（sys_roles.code 全局目录 ×
// sys_user_roles 用户分配）。审批级角色校验用。
func (r *ApprovalFlowRepo) UserHoldsRole(ctx context.Context, userID uint32, roleCode string) (bool, error) {
	// 角色目录是平台级全局表（tenant_id=0），租户上下文直查会被
	// TenantPrivacy 过滤成空集——切系统视图（镜像 permission_repo）。
	roleIDs, err := r.entClient.Client().Role.Query().
		Where(role.CodeEQ(roleCode)).
		IDs(appViewer.NewSystemViewerContext(ctx))
	if err != nil || len(roleIDs) == 0 {
		return false, nil
	}
	count, err := r.entClient.Client().UserRole.Query().
		Where(
			userrole.UserIDEQ(userID),
			userrole.RoleIDIn(roleIDs...),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("query user role failed: %s", err.Error())
		return false, approvalV1.ErrorInternalServerError("query user role failed")
	}
	return count > 0, nil
}
