package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type WarehouseService struct {
	appV1.WarehouseServiceHTTPServer

	warehouseServiceClient inventoryV1.WarehouseServiceClient

	log *log.Helper
}

func NewWarehouseService(
	ctx *bootstrap.Context,
	warehouseServiceClient inventoryV1.WarehouseServiceClient,
) *WarehouseService {
	return &WarehouseService{
		log:                   ctx.NewLoggerHelper("warehouse/service/app-service"),
		warehouseServiceClient: warehouseServiceClient,
	}
}

func (s *WarehouseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListWarehouseResponse, error) {
	return s.warehouseServiceClient.List(ctx, req)
}

func (s *WarehouseService) Get(ctx context.Context, req *inventoryV1.GetWarehouseRequest) (*inventoryV1.Warehouse, error) {
	return s.warehouseServiceClient.Get(ctx, req)
}
