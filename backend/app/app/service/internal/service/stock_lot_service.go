package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockLotService 批次库存（app BFF facade，移动端只读）。
type StockLotService struct {
	appV1.StockLotServiceHTTPServer

	stockLotServiceClient inventoryV1.StockLotServiceClient

	log *log.Helper
}

func NewStockLotService(
	ctx *bootstrap.Context,
	stockLotServiceClient inventoryV1.StockLotServiceClient,
) *StockLotService {
	return &StockLotService{
		stockLotServiceClient: stockLotServiceClient,
		log:                   ctx.NewLoggerHelper("stock_lot/service/app-service"),
	}
}

func (s *StockLotService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockLotResponse, error) {
	return s.stockLotServiceClient.List(ctx, req)
}
