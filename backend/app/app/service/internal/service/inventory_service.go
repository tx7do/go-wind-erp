package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type InventoryService struct {
	appV1.InventoryServiceHTTPServer

	inventoryServiceClient inventoryV1.InventoryServiceClient

	log *log.Helper
}

func NewInventoryService(
	ctx *bootstrap.Context,
	inventoryServiceClient inventoryV1.InventoryServiceClient,
) *InventoryService {
	return &InventoryService{
		log:                    ctx.NewLoggerHelper("inventory/service/app-service"),
		inventoryServiceClient: inventoryServiceClient,
	}
}

func (s *InventoryService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListInventoryResponse, error) {
	return s.inventoryServiceClient.List(ctx, req)
}

func (s *InventoryService) Get(ctx context.Context, req *inventoryV1.GetInventoryRequest) (*inventoryV1.Inventory, error) {
	return s.inventoryServiceClient.Get(ctx, req)
}

func (s *InventoryService) GetOverview(ctx context.Context, req *inventoryV1.GetInventoryOverviewRequest) (*inventoryV1.InventoryOverview, error) {
	return s.inventoryServiceClient.GetOverview(ctx, req)
}
