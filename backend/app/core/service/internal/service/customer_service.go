package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

// CustomerService 客户服务
type CustomerService struct {
	salesV1.UnimplementedCustomerServiceServer

	log *log.Helper

	customerRepo *data.CustomerRepo
}

func NewCustomerService(
	ctx *bootstrap.Context,
	customerRepo *data.CustomerRepo,
) *CustomerService {
	svc := &CustomerService{
		log:          ctx.NewLoggerHelper("customer/service/core-service"),
		customerRepo: customerRepo,
	}

	return svc
}

func (s *CustomerService) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListCustomerResponse, error) {
	return s.customerRepo.List(ctx, req)
}

func (s *CustomerService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.CountCustomerResponse, error) {
	count, err := s.customerRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &salesV1.CountCustomerResponse{Count: uint64(count)}, nil
}

func (s *CustomerService) Get(ctx context.Context, req *salesV1.GetCustomerRequest) (*salesV1.Customer, error) {
	return s.customerRepo.Get(ctx, req)
}

func (s *CustomerService) Create(ctx context.Context, req *salesV1.CreateCustomerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	if err := validateCustomerFields(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.customerRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *CustomerService) Update(ctx context.Context, req *salesV1.UpdateCustomerRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	if err := validateCustomerFields(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.customerRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// validateCustomerFields 校验客户必填字段。
func validateCustomerFields(v *salesV1.Customer) error {
	if v == nil {
		return salesV1.ErrorBadRequest("invalid parameter")
	}
	if v.GetCode() == "" {
		return salesV1.ErrorBadRequest("code is required")
	}
	if v.GetName() == "" {
		return salesV1.ErrorBadRequest("name is required")
	}
	return nil
}

func (s *CustomerService) Delete(ctx context.Context, req *salesV1.DeleteCustomerRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid request")
	}

	if err := s.customerRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
