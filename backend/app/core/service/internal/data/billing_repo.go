package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/plan"
	"go-wind-erp/app/core/service/internal/data/ent/subscription"

	billingV1 "go-wind-erp/api/gen/go/billing/service/v1"

	appViewer "go-wind-erp/pkg/entgo/viewer"
)

// PlanRepo 套餐目录仓储。套餐是平台级全局目录（tenant_id=0）——
// 所有查询切系统视图（镜像 sys_permissions 的目录语义），否则
// TenantPrivacy 会给租户上下文注入 tenant_id 过滤得到空集。
type PlanRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[billingV1.Plan, ent.Plan]
}

func NewPlanRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PlanRepo {
	repo := &PlanRepo{
		log:       ctx.NewLoggerHelper("plan/repo/core-service"),
		entClient: entClient,
	}
	repo.mapper = mapper.NewCopierMapper[billingV1.Plan, ent.Plan]()
	return repo
}

// List 套餐列表（系统视图，按 sort_order 排序，定价页/管理端共用）。
func (r *PlanRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.ListPlanResponse, error) {
	ctx = appViewer.NewSystemViewerContext(ctx)
	entities, err := r.entClient.Client().Plan.Query().
		Order(ent.Asc(plan.FieldSortOrder)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list plans failed: %s", err.Error())
		return nil, billingV1.ErrorInternalServerError("list plans failed")
	}

	dtos := make([]*billingV1.Plan, 0, len(entities))
	for _, e := range entities {
		dtos = append(dtos, r.mapper.ToDTO(e))
	}
	return &billingV1.ListPlanResponse{
		Total: uint64(len(dtos)),
		Items: dtos,
	}, nil
}

// GetByCode 按编码取套餐（守卫/切换套餐用；系统视图）。未找到返回 nil, nil。
func (r *PlanRepo) GetByCode(ctx context.Context, code string) (*billingV1.Plan, error) {
	if code == "" {
		return nil, nil
	}
	ctx = appViewer.NewSystemViewerContext(ctx)
	entity, err := r.entClient.Client().Plan.Query().
		Where(plan.CodeEQ(code)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, billingV1.ErrorInternalServerError("query plan failed")
	}
	return r.mapper.ToDTO(entity), nil
}

// Create 新建套餐（系统视图写入平台目录）。
func (r *PlanRepo) Create(ctx context.Context, req *billingV1.CreatePlanRequest) error {
	if req == nil || req.Data == nil {
		return billingV1.ErrorBadRequest("invalid parameter")
	}
	ctx = appViewer.NewSystemViewerContext(ctx)
	builder := r.entClient.Client().Plan.Create().
		SetNillableCode(req.Data.Code).
		SetNillableName(req.Data.Name).
		SetNillableDescription(req.Data.Description).
		SetNillablePriceCents(req.Data.PriceCents).
		SetNillableMaxUsers(req.Data.MaxUsers).
		SetNillableMaxOrdersMonthly(req.Data.MaxOrdersMonthly).
		SetNillableSortOrder(req.Data.SortOrder)
	return builder.Exec(ctx)
}

// Update 更新套餐（系统视图）。
func (r *PlanRepo) Update(ctx context.Context, req *billingV1.UpdatePlanRequest) error {
	if req == nil || req.Data == nil {
		return billingV1.ErrorBadRequest("invalid parameter")
	}
	ctx = appViewer.NewSystemViewerContext(ctx)
	builder := r.entClient.Client().Plan.Update().
		Where(plan.IDEQ(req.Data.GetId()))
	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		if containsPath(req.UpdateMask.Paths, "name") {
			builder.SetNillableName(req.Data.Name)
		}
		if containsPath(req.UpdateMask.Paths, "description") {
			builder.SetNillableDescription(req.Data.Description)
		}
		if containsPath(req.UpdateMask.Paths, "priceCents") {
			builder.SetNillablePriceCents(req.Data.PriceCents)
		}
		if containsPath(req.UpdateMask.Paths, "maxUsers") {
			builder.SetNillableMaxUsers(req.Data.MaxUsers)
		}
		if containsPath(req.UpdateMask.Paths, "maxOrdersMonthly") {
			builder.SetNillableMaxOrdersMonthly(req.Data.MaxOrdersMonthly)
		}
		if containsPath(req.UpdateMask.Paths, "sortOrder") {
			builder.SetNillableSortOrder(req.Data.SortOrder)
		}
	} else {
		builder.
			SetNillableName(req.Data.Name).
			SetNillableDescription(req.Data.Description).
			SetNillablePriceCents(req.Data.PriceCents).
			SetNillableMaxUsers(req.Data.MaxUsers).
			SetNillableMaxOrdersMonthly(req.Data.MaxOrdersMonthly).
			SetNillableSortOrder(req.Data.SortOrder)
	}
	return builder.Exec(ctx)
}

// Delete 删除套餐（系统视图）。
func (r *PlanRepo) Delete(ctx context.Context, req *billingV1.DeletePlanRequest) error {
	if req == nil || req.GetId() == 0 {
		return billingV1.ErrorBadRequest("invalid parameter")
	}
	ctx = appViewer.NewSystemViewerContext(ctx)
	return r.entClient.Client().Plan.DeleteOneID(req.GetId()).Exec(ctx)
}

func containsPath(paths []string, p string) bool {
	for _, v := range paths {
		if v == p {
			return true
		}
	}
	return false
}

// SubscriptionRepo 租户订阅仓储（租户作用域，每租户一条当前订阅）。
type SubscriptionRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[billingV1.Subscription, ent.Subscription]
}

func NewSubscriptionRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *SubscriptionRepo {
	return &SubscriptionRepo{
		log:       ctx.NewLoggerHelper("subscription/repo/core-service"),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[billingV1.Subscription, ent.Subscription](),
	}
}

// GetByTenant 取租户当前订阅（租户上下文）。无记录返回 nil, nil
// （守卫按 FREE 默认限额处理）。
func (r *SubscriptionRepo) GetByTenant(ctx context.Context, tenantID uint32) (*billingV1.Subscription, error) {
	if tenantID == 0 {
		return nil, nil
	}
	entity, err := r.entClient.Client().Subscription.Query().
		Where(subscription.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, billingV1.ErrorInternalServerError("query subscription failed")
	}
	// 手动映射时间戳与状态（CopierMapper 不能自动转换 time.Time ↔ timestamppb）。
	dto := &billingV1.Subscription{
		Id:       &entity.ID,
		TenantId: entity.TenantID,
		PlanCode: entity.PlanCode,
	}
	if entity.PeriodStart != nil {
		dto.PeriodStart = timestamppb.New(*entity.PeriodStart)
	}
	if entity.PeriodEnd != nil {
		dto.PeriodEnd = timestamppb.New(*entity.PeriodEnd)
	}
	if entity.Status != nil {
		dto.Status = billingV1.Subscription_Status(billingV1.Subscription_Status_value[string(*entity.Status)]).Enum()
	}
	return dto, nil
}

// UpsertByTenant 创建或覆盖租户订阅（切套餐/续费/注册挂 FREE 共用；
// 每租户一条，唯一索引兜底并发）。
func (r *SubscriptionRepo) UpsertByTenant(
	ctx context.Context,
	tenantID uint32,
	sub *billingV1.Subscription,
) error {
	if tenantID == 0 || sub == nil {
		return billingV1.ErrorBadRequest("invalid parameter")
	}

	existing, err := r.entClient.Client().Subscription.Query().
		Where(subscription.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return billingV1.ErrorInternalServerError("query subscription failed")
	}

	if existing == nil {
		builder := r.entClient.Client().Subscription.Create().
			SetTenantID(tenantID).
			SetNillablePlanCode(sub.PlanCode).
			SetNillablePeriodStart(protoTime(sub.PeriodStart)).
			SetNillablePeriodEnd(protoTime(sub.PeriodEnd))
		if sub.Status != nil {
			builder.SetStatus(subscription.Status(sub.Status.String()))
		}
		return builder.Exec(ctx)
	}

	builder := r.entClient.Client().Subscription.Update().
		Where(subscription.IDEQ(existing.ID)).
		SetNillablePlanCode(sub.PlanCode).
		SetNillablePeriodStart(protoTime(sub.PeriodStart)).
		SetNillablePeriodEnd(protoTime(sub.PeriodEnd))
	if sub.Status != nil {
		builder.SetStatus(subscription.Status(sub.Status.String()))
	}
	return builder.Exec(ctx)
}

