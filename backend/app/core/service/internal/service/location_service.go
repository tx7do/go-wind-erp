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

// LocationService 库存位置服务（借鉴 Odoo stock.location）。
type LocationService struct {
	inventoryV1.UnimplementedLocationServiceServer

	log *log.Helper

	locationRepo *data.LocationRepo
}

func NewLocationService(
	ctx *bootstrap.Context,
	locationRepo *data.LocationRepo,
) *LocationService {
	svc := &LocationService{
		log:          ctx.NewLoggerHelper("location/service/core-service"),
		locationRepo: locationRepo,
	}

	return svc
}

func (s *LocationService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListLocationResponse, error) {
	return s.locationRepo.List(ctx, req)
}

func (s *LocationService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountLocationResponse, error) {
	count, err := s.locationRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountLocationResponse{Count: uint64(count)}, nil
}

func (s *LocationService) Get(ctx context.Context, req *inventoryV1.GetLocationRequest) (*inventoryV1.StockLocation, error) {
	return s.locationRepo.Get(ctx, req)
}

func (s *LocationService) Create(ctx context.Context, req *inventoryV1.CreateLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.locationRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *LocationService) Update(ctx context.Context, req *inventoryV1.UpdateLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.locationRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *LocationService) Delete(ctx context.Context, req *inventoryV1.DeleteLocationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	if err := s.locationRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
