package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

// CustomerService 客户管理（admin BFF facade）
type CustomerService struct {
	adminV1.CustomerServiceHTTPServer

	log *log.Helper

	customerServiceClient salesV1.CustomerServiceClient
}

func NewCustomerService(
	ctx *bootstrap.Context,
	customerServiceClient salesV1.CustomerServiceClient,
) *CustomerService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "customer/service/admin-service"))
	return &CustomerService{
		log:                   l,
		customerServiceClient: customerServiceClient,
	}
}

func (s *CustomerService) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListCustomerResponse, error) {
	return s.customerServiceClient.List(ctx, req)
}

func (s *CustomerService) Get(ctx context.Context, req *salesV1.GetCustomerRequest) (*salesV1.Customer, error) {
	return s.customerServiceClient.Get(ctx, req)
}

func (s *CustomerService) Create(ctx context.Context, req *salesV1.CreateCustomerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.customerServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *CustomerService) Update(ctx context.Context, req *salesV1.UpdateCustomerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	_, err = s.customerServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *CustomerService) Delete(ctx context.Context, req *salesV1.DeleteCustomerRequest) (*emptypb.Empty, error) {
	return s.customerServiceClient.Delete(ctx, req)
}

// SalesOrderService 销售单管理（admin BFF facade）
type SalesOrderService struct {
	adminV1.SalesOrderServiceHTTPServer

	log *log.Helper

	salesOrderServiceClient salesV1.SalesOrderServiceClient
}

func NewSalesOrderService(
	ctx *bootstrap.Context,
	salesOrderServiceClient salesV1.SalesOrderServiceClient,
) *SalesOrderService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "sales_order/service/admin-service"))
	return &SalesOrderService{
		log:                    l,
		salesOrderServiceClient: salesOrderServiceClient,
	}
}

func (s *SalesOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListSalesOrderResponse, error) {
	return s.salesOrderServiceClient.List(ctx, req)
}

func (s *SalesOrderService) Get(ctx context.Context, req *salesV1.GetSalesOrderRequest) (*salesV1.SalesOrder, error) {
	return s.salesOrderServiceClient.Get(ctx, req)
}

func (s *SalesOrderService) Create(ctx context.Context, req *salesV1.CreateSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.salesOrderServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *SalesOrderService) Update(ctx context.Context, req *salesV1.UpdateSalesOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	_, err = s.salesOrderServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *SalesOrderService) Delete(ctx context.Context, req *salesV1.DeleteSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Delete(ctx, req)
}

// 动作类（submit/approve/reject/cancel/complete）：纯委派 core，
// 发起人与自审守卫由 core 的 viewer context 推导。
func (s *SalesOrderService) Submit(ctx context.Context, req *salesV1.SubmitSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Submit(ctx, req)
}

func (s *SalesOrderService) Approve(ctx context.Context, req *salesV1.ApproveSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Approve(ctx, req)
}

func (s *SalesOrderService) Reject(ctx context.Context, req *salesV1.RejectSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Reject(ctx, req)
}

func (s *SalesOrderService) Cancel(ctx context.Context, req *salesV1.CancelSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Cancel(ctx, req)
}

func (s *SalesOrderService) Complete(ctx context.Context, req *salesV1.CompleteSalesOrderRequest) (*emptypb.Empty, error) {
	return s.salesOrderServiceClient.Complete(ctx, req)
}
