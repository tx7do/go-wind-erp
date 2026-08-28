package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-crud/viewer"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-erp/app/core/service/internal/data"

	billingV1 "go-wind-erp/api/gen/go/billing/service/v1"

	"go-wind-erp/pkg/constants"
)

// BillingGuard 透明定价的配额守卫：在单据/用户创建前校验订阅限额。
//
// 语义（免费+按用户订阅，到期只读）：
//   - 平台上下文（tid=0）跳过——平台管理员不受租户配额限制
//   - 无订阅记录 → 按 FREE 默认限额处理（admin 建的租户与自助注册等效）
//   - period_end 已过 → 409（数据保留只读，续费恢复）
//   - 本月 PO+SO 创建数 ≥ max_orders → 409（升级恢复）
//   - 租户用户数 ≥ max_users → 409（升级恢复）
//
// 0 = 无限。SAGA 自动补货草稿不走守卫（系统自动行为不卡配额）。
type BillingGuard struct {
	log *log.Helper

	subscriptionRepo *data.SubscriptionRepo
	planRepo         *data.PlanRepo
	userRepo         data.UserRepo
	purchaseRepo     *data.PurchaseOrderRepo
	salesRepo        *data.SalesOrderRepo
}

func NewBillingGuard(
	ctx *bootstrap.Context,
	subscriptionRepo *data.SubscriptionRepo,
	planRepo *data.PlanRepo,
	userRepo data.UserRepo,
	purchaseRepo *data.PurchaseOrderRepo,
	salesRepo *data.SalesOrderRepo,
) *BillingGuard {
	return &BillingGuard{
		log:              ctx.NewLoggerHelper("billing/guard/core-service"),
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		userRepo:         userRepo,
		purchaseRepo:     purchaseRepo,
		salesRepo:        salesRepo,
	}
}

func viewerTenantID(ctx context.Context) (uint32, bool) {
	vc, exist := viewer.FromContext(ctx)
	if !exist || vc == nil {
		return 0, false
	}
	tid := vc.TenantID()
	if tid == 0 || !vc.IsTenantContext() {
		return 0, false
	}
	return uint32(tid), true
}

// currentPlan 取租户当前生效的套餐 + 订阅。无订阅 → FREE 套餐（或内置默认）。
func (g *BillingGuard) currentPlan(ctx context.Context, tenantID uint32) (*billingV1.Plan, *billingV1.Subscription) {
	sub, _ := g.subscriptionRepo.GetByTenant(ctx, tenantID)

	planCode := constants.FreePlanCode
	if sub != nil && sub.GetPlanCode() != "" {
		planCode = sub.GetPlanCode()
	}

	// 套餐目录查询切系统视图（平台级目录，GetByCode 内部处理）。
	plan, _ := g.planRepo.GetByCode(ctx, planCode)
	if plan == nil {
		// 目录缺种子的兜底：内置 FREE 默认（3 用户 / 100 单/月）。
		plan = &billingV1.Plan{
			Code:             &planCode,
			MaxUsers:         int64Ptr(3),
			MaxOrdersMonthly: int64Ptr(100),
		}
	}
	return plan, sub
}

// EnsureCanCreateOrder 校验订阅未过期 + 本月单量未超限。
func (g *BillingGuard) EnsureCanCreateOrder(ctx context.Context) error {
	tid, ok := viewerTenantID(ctx)
	if !ok {
		return nil // 平台上下文不受限
	}

	plan, sub := g.currentPlan(ctx, tid)

	if sub != nil && sub.GetPeriodEnd() != nil {
		if time.Now().After(sub.GetPeriodEnd().AsTime()) {
			return billingV1.ErrorConflict(
				"订阅已到期，数据已保留（只读），续费后即可继续开单")
		}
	}

	maxOrders := plan.GetMaxOrdersMonthly()
	if maxOrders > 0 {
		monthStart := time.Now().AddDate(0, 0, -time.Now().Day()+1)
		poN, err := g.purchaseRepo.CountTenantSince(ctx, tid, monthStart)
		if err != nil {
			return billingV1.ErrorInternalServerError("quota check failed")
		}
		soN, err := g.salesRepo.CountTenantSince(ctx, tid, monthStart)
		if err != nil {
			return billingV1.ErrorInternalServerError("quota check failed")
		}
		if int64(poN+soN) >= maxOrders {
			return billingV1.ErrorConflict(
				"本月单量已达上限（%d 张），升级套餐后继续", maxOrders)
		}
	}
	return nil
}

// EnsureCanCreateUser 校验订阅未过期 + 租户用户数未超限。
func (g *BillingGuard) EnsureCanCreateUser(ctx context.Context) error {
	tid, ok := viewerTenantID(ctx)
	if !ok {
		return nil // 平台上下文不受限
	}

	plan, sub := g.currentPlan(ctx, tid)

	if sub != nil && sub.GetPeriodEnd() != nil {
		if time.Now().After(sub.GetPeriodEnd().AsTime()) {
			return billingV1.ErrorConflict(
				"订阅已到期，数据已保留（只读），续费后即可继续添加用户")
		}
	}

	maxUsers := plan.GetMaxUsers()
	if maxUsers > 0 {
		n, err := g.userRepo.CountByTenant(ctx, tid)
		if err != nil {
			return billingV1.ErrorInternalServerError("quota check failed")
		}
		if int64(n) >= maxUsers {
			return billingV1.ErrorConflict(
				"用户数已达上限（%d 个），升级套餐后继续", maxUsers)
		}
	}
	return nil
}

func int64Ptr(v int64) *int64 { return &v }
