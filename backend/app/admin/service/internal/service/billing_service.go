package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	billingV1 "go-wind-erp/api/gen/go/billing/service/v1"
)

// BillingService 计费服务（admin BFF facade，租户自助）。
type BillingService struct {
	adminV1.BillingServiceHTTPServer

	log *log.Helper

	billingServiceClient billingV1.BillingServiceClient
}

func NewBillingService(
	ctx *bootstrap.Context,
	billingServiceClient billingV1.BillingServiceClient,
) *BillingService {
	return &BillingService{
		log:                  ctx.NewLoggerHelper("billing/service/admin-service"),
		billingServiceClient: billingServiceClient,
	}
}

func (s *BillingService) GetMySubscription(ctx context.Context, req *emptypb.Empty) (*billingV1.SubscriptionUsage, error) {
	return s.billingServiceClient.GetMySubscription(ctx, req)
}

func (s *BillingService) ListPlans(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.ListPlanResponse, error) {
	return s.billingServiceClient.ListPlans(ctx, req)
}

func (s *BillingService) ChangePlan(ctx context.Context, req *billingV1.ChangePlanRequest) (*emptypb.Empty, error) {
	return s.billingServiceClient.ChangePlan(ctx, req)
}

func (s *BillingService) Renew(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return s.billingServiceClient.Renew(ctx, req)
}

// PlanAdminService 套餐管理（admin BFF facade，平台管理员）。
type PlanAdminService struct {
	adminV1.PlanAdminServiceHTTPServer

	log *log.Helper

	planAdminServiceClient billingV1.PlanAdminServiceClient
}

func NewPlanAdminService(
	ctx *bootstrap.Context,
	planAdminServiceClient billingV1.PlanAdminServiceClient,
) *PlanAdminService {
	return &PlanAdminService{
		log:                   ctx.NewLoggerHelper("plan_admin/service/admin-service"),
		planAdminServiceClient: planAdminServiceClient,
	}
}

func (s *PlanAdminService) List(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.ListPlanResponse, error) {
	return s.planAdminServiceClient.List(ctx, req)
}

func (s *PlanAdminService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*billingV1.CountPlanResponse, error) {
	return s.planAdminServiceClient.Count(ctx, req)
}

func (s *PlanAdminService) Get(ctx context.Context, req *billingV1.GetPlanRequest) (*billingV1.Plan, error) {
	return s.planAdminServiceClient.Get(ctx, req)
}

func (s *PlanAdminService) Create(ctx context.Context, req *billingV1.CreatePlanRequest) (*emptypb.Empty, error) {
	return s.planAdminServiceClient.Create(ctx, req)
}

func (s *PlanAdminService) Update(ctx context.Context, req *billingV1.UpdatePlanRequest) (*emptypb.Empty, error) {
	return s.planAdminServiceClient.Update(ctx, req)
}

func (s *PlanAdminService) Delete(ctx context.Context, req *billingV1.DeletePlanRequest) (*emptypb.Empty, error) {
	return s.planAdminServiceClient.Delete(ctx, req)
}
