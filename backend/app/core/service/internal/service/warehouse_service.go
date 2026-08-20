package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// WarehouseService 仓库服务
type WarehouseService struct {
	inventoryV1.UnimplementedWarehouseServiceServer

	log *log.Helper

	warehouseRepo *data.WarehouseRepo
}

func NewWarehouseService(
	ctx *bootstrap.Context,
	warehouseRepo *data.WarehouseRepo,
) *WarehouseService {
	svc := &WarehouseService{
		log:      ctx.NewLoggerHelper("warehouse/service/core-service"),
		warehouseRepo: warehouseRepo,
	}

	return svc
}

func (s *WarehouseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListWarehouseResponse, error) {
	return s.warehouseRepo.List(ctx, req)
}

func (s *WarehouseService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountWarehouseResponse, error) {
	count, err := s.warehouseRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountWarehouseResponse{Count: uint64(count)}, nil
}

func (s *WarehouseService) Get(ctx context.Context, req *inventoryV1.GetWarehouseRequest) (*inventoryV1.Warehouse, error) {
	return s.warehouseRepo.Get(ctx, req)
}

func (s *WarehouseService) Create(ctx context.Context, req *inventoryV1.CreateWarehouseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.warehouseRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *WarehouseService) Update(ctx context.Context, req *inventoryV1.UpdateWarehouseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.warehouseRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *WarehouseService) Delete(ctx context.Context, req *inventoryV1.DeleteWarehouseRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	if err := s.warehouseRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
