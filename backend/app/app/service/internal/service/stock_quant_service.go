package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type StockQuantService struct {
	appV1.StockQuantServiceHTTPServer

	stockQuantServiceClient inventoryV1.StockQuantServiceClient

	log *log.Helper
}

func NewStockQuantService(
	ctx *bootstrap.Context,
	stockQuantServiceClient inventoryV1.StockQuantServiceClient,
) *StockQuantService {
	return &StockQuantService{
		log:                      ctx.NewLoggerHelper("stock_quant/service/app-service"),
		stockQuantServiceClient: stockQuantServiceClient,
	}
}

func (s *StockQuantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockQuantResponse, error) {
	return s.stockQuantServiceClient.List(ctx, req)
}

func (s *StockQuantService) Get(ctx context.Context, req *inventoryV1.GetStockQuantRequest) (*inventoryV1.StockQuant, error) {
	return s.stockQuantServiceClient.Get(ctx, req)
}

func (s *StockQuantService) GetOverview(ctx context.Context, req *inventoryV1.GetStockQuantOverviewRequest) (*inventoryV1.StockQuantOverview, error) {
	return s.stockQuantServiceClient.GetOverview(ctx, req)
}
