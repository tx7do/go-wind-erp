package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockLotService 批次库存（admin BFF facade，纯委派 core）。
type StockLotService struct {
	adminV1.StockLotServiceHTTPServer

	log *log.Helper

	stockLotServiceClient inventoryV1.StockLotServiceClient
}

func NewStockLotService(
	ctx *bootstrap.Context,
	stockLotServiceClient inventoryV1.StockLotServiceClient,
) *StockLotService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "stock-lot/service/admin-service"))
	return &StockLotService{
		log:                    l,
		stockLotServiceClient: stockLotServiceClient,
	}
}

func (s *StockLotService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockLotResponse, error) {
	return s.stockLotServiceClient.List(ctx, req)
}
