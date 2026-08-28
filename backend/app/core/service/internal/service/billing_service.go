package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data"

	billingV1 "go-wind-erp/api/gen/go/billing/service/v1"

	"go-wind-erp/pkg/constants"
	appViewer "go-wind-erp/pkg/entgo/viewer"
)

// 订阅周期：付费套餐按月（30 天）。
const subscriptionPeriodDays = 30

// BillingService 计费服务（透明定价）：套餐目录种子 + 租户自助查订阅/切套餐/续费。
// 真实支付网关不在范围——切换套餐为线下付款后自助激活。
type BillingService struct {
	billingV1.UnimplementedBillingServiceServer

	log *log.Helper

	planRepo         *data.PlanRepo
	subscriptionRepo *data.SubscriptionRepo
	userRepo         data.UserRepo
	purchaseRepo     *data.PurchaseOrderRepo
	salesRepo        *data.SalesOrderRepo
}

func NewBillingService(
	ctx *bootstrap.Context,
	planRepo *data.PlanRepo,
	subscriptionRepo *data.SubscriptionRepo,
	userRepo data.UserRepo,
	purchaseRepo *data.PurchaseOrderRepo,
	salesRepo *data.SalesOrderRepo,
) *BillingService {
	svc := &BillingService{
		log:              ctx.NewLoggerHelper("billing/service/core-service"),
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		userRepo:         userRepo,
		purchaseRepo:     purchaseRepo,
		salesRepo:        salesRepo,
	}
	svc.init()
	return svc
}

// init 种子默认套餐（镜像 permission_service 的目录种子模式）。
func (s *BillingService) init() {
	ctx := appViewer.NewSystemViewerContext(context.Background())
	if resp, _ := s.planRepo.List(ctx, nil); resp != nil && len(resp.Items) > 0 {
		return
	}
	for _, p := range constants.DefaultPlans {
		if err := s.planRepo.Create(ctx, &billingV1.CreatePlanRequest{Data: p}); err != nil {
			s.log.Errorf("seed plan %s failed: %v", p.GetCode(), err)
		}
	}
	s.log.Infof("billing: seeded %d default plans", len(constants.DefaultPlans))
}

// GetMySubscription 当前租户订阅 + 套餐 + 实时用量（前端进度条数据源）。
func (s *BillingService) GetMySubscription(ctx context.Context, _ *emptypb.Empty) (*billingV1.SubscriptionUsage, error) {
	guard := &BillingGuard{
		log:              s.log,
		subscriptionRepo: s.subscriptionRepo,
		planRepo:         s.planRepo,
		userRepo:         s.userRepo,
		purchaseRepo:     s.purchaseRepo,
		salesRepo:        s.salesRepo,
	}

	tid, ok := viewerTenantID(ctx)
	if !ok {
		return nil, billingV1.ErrorBadRequest("tenant context required")
	}

	plan, sub := guard.currentPlan(ctx, tid)

	userCount, _ := s.userRepo.CountByTenant(ctx, tid)

	monthStart := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	poN, _ := s.purchaseRepo.CountTenantSince(ctx, tid, monthStart)
	soN, _ := s.salesRepo.CountTenantSince(ctx, tid, monthStart)

	expired := false
	if sub != nil && sub.GetPeriodEnd() != nil {
		expired = time.Now().After(sub.GetPeriodEnd().AsTime())
	}

	return &billingV1.SubscriptionUsage{
		Subscription:    sub,
		Plan:            plan,
		UserCount:       trans.Ptr(int64(userCount)),
		OrderCountMonth: trans.Ptr(int64(poN + soN)),
		Expired:         &expired,
	}, nil
}

// ListPlans 公开套餐列表（定价页）。
func (s *BillingService) ListPlans(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.ListPlanResponse, error) {
	return s.planRepo.List(ctx, req)
}

// ChangePlan 切换套餐：免费→付费=开通30天；付费→免费=降级（永久）；
// 付费→付费=变更并重置30天周期。
func (s *BillingService) ChangePlan(ctx context.Context, req *billingV1.ChangePlanRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetPlanCode() == "" {
		return nil, billingV1.ErrorBadRequest("plan_code is required")
	}

	tid, ok := viewerTenantID(ctx)
	if !ok {
		return nil, billingV1.ErrorBadRequest("tenant context required")
	}

	plan, _ := s.planRepo.GetByCode(ctx, req.GetPlanCode())
	if plan == nil {
		return nil, billingV1.ErrorNotFound("plan not found")
	}

	now := time.Now()
	sub := &billingV1.Subscription{
		PlanCode: &req.PlanCode,
		Status:   billingV1.Subscription_ACTIVE.Enum(),
	}

	if req.GetPlanCode() == constants.FreePlanCode {
		// 降级到免费：无到期（永久）。
		sub.PeriodStart = timestamppb.New(now)
		sub.PeriodEnd = nil
	} else {
		// 开通/变更付费：30 天周期。
		sub.PeriodStart = timestamppb.New(now)
		sub.PeriodEnd = timestamppb.New(now.AddDate(0, 0, subscriptionPeriodDays))
	}

	if err := s.subscriptionRepo.UpsertByTenant(ctx, tid, sub); err != nil {
		return nil, err
	}

	s.log.Infof("billing: tenant %d changed plan to %s", tid, req.GetPlanCode())
	return &emptypb.Empty{}, nil
}

// Renew 续费：当前付费套餐延长30天（从当前 period_end 或 now 起算，取晚者）。
func (s *BillingService) Renew(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	tid, ok := viewerTenantID(ctx)
	if !ok {
		return nil, billingV1.ErrorBadRequest("tenant context required")
	}

	sub, err := s.subscriptionRepo.GetByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.GetPlanCode() == constants.FreePlanCode {
		return nil, billingV1.ErrorBadRequest("current plan is FREE; upgrade first")
	}

	now := time.Now()
	base := now
	if sub.GetPeriodEnd() != nil && sub.GetPeriodEnd().AsTime().After(now) {
		base = sub.GetPeriodEnd().AsTime()
	}

	updated := &billingV1.Subscription{
		PlanCode:    sub.PlanCode,
		PeriodStart: sub.PeriodStart,
		PeriodEnd:   timestamppb.New(base.AddDate(0, 0, subscriptionPeriodDays)),
		Status:      billingV1.Subscription_ACTIVE.Enum(),
	}
	if err := s.subscriptionRepo.UpsertByTenant(ctx, tid, updated); err != nil {
		return nil, err
	}

	s.log.Infof("billing: tenant %d renewed plan %s", tid, sub.GetPlanCode())
	return &emptypb.Empty{}, nil
}

// PlanAdminService 套餐管理（平台管理员 CRUD）。
type PlanAdminService struct {
	billingV1.UnimplementedPlanAdminServiceServer

	log *log.Helper

	planRepo *data.PlanRepo
}

func NewPlanAdminService(
	ctx *bootstrap.Context,
	planRepo *data.PlanRepo,
) *PlanAdminService {
	return &PlanAdminService{
		log:      ctx.NewLoggerHelper("plan_admin/service/core-service"),
		planRepo: planRepo,
	}
}

func (s *PlanAdminService) List(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.ListPlanResponse, error) {
	return s.planRepo.List(ctx, req)
}

func (s *PlanAdminService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.CountPlanResponse, error) {
	resp, err := s.planRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	return &billingV1.CountPlanResponse{Count: int64(resp.GetTotal())}, nil
}

func (s *PlanAdminService) Get(ctx context.Context, req *billingV1.GetPlanRequest) (*billingV1.Plan, error) {
	resp, err := s.planRepo.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	for _, p := range resp.GetItems() {
		if p.GetId() == req.GetId() {
			return p, nil
		}
	}
	return nil, billingV1.ErrorNotFound("plan not found")
}

func (s *PlanAdminService) Create(ctx context.Context, req *billingV1.CreatePlanRequest) (*emptypb.Empty, error) {
	if err := s.planRepo.Create(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *PlanAdminService) Update(ctx context.Context, req *billingV1.UpdatePlanRequest) (*emptypb.Empty, error) {
	if err := s.planRepo.Update(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *PlanAdminService) Delete(ctx context.Context, req *billingV1.DeletePlanRequest) (*emptypb.Empty, error) {
	if err := s.planRepo.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
