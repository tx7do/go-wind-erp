package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// SupplierService 供应商服务
type SupplierService struct {
	procurementV1.UnimplementedSupplierServiceServer

	log *log.Helper

	supplierRepo *data.SupplierRepo
}

func NewSupplierService(
	ctx *bootstrap.Context,
	supplierRepo *data.SupplierRepo,
) *SupplierService {
	svc := &SupplierService{
		log:         ctx.NewLoggerHelper("supplier/service/core-service"),
		supplierRepo: supplierRepo,
	}

	return svc
}

func (s *SupplierService) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListSupplierResponse, error) {
	return s.supplierRepo.List(ctx, req)
}

func (s *SupplierService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.CountSupplierResponse, error) {
	count, err := s.supplierRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &procurementV1.CountSupplierResponse{Count: uint64(count)}, nil
}

func (s *SupplierService) Get(ctx context.Context, req *procurementV1.GetSupplierRequest) (*procurementV1.Supplier, error) {
	return s.supplierRepo.Get(ctx, req)
}

func (s *SupplierService) Create(ctx context.Context, req *procurementV1.CreateSupplierRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	if err := validateSupplierFields(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.supplierRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *SupplierService) Update(ctx context.Context, req *procurementV1.UpdateSupplierRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	if err := validateSupplierFields(req.Data); err != nil {
		return nil, err
	}

	if _, err := s.supplierRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// validateSupplierFields 校验供应商必填字段。
func validateSupplierFields(v *procurementV1.Supplier) error {
	if v == nil {
		return procurementV1.ErrorBadRequest("invalid parameter")
	}
	if v.GetCode() == "" {
		return procurementV1.ErrorBadRequest("code is required")
	}
	if v.GetName() == "" {
		return procurementV1.ErrorBadRequest("name is required")
	}
	return nil
}

func (s *SupplierService) Delete(ctx context.Context, req *procurementV1.DeleteSupplierRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid request")
	}

	if err := s.supplierRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
