package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

// SalesOrderService 销售单（app BFF facade，移动端只读：
// 销售在路上查单据状态与履约进度；退货走 StockPickingService.CreateSalesReturn）。
type SalesOrderService struct {
	appV1.SalesOrderServiceHTTPServer

	salesOrderServiceClient salesV1.SalesOrderServiceClient

	log *log.Helper
}

func NewSalesOrderService(
	ctx *bootstrap.Context,
	salesOrderServiceClient salesV1.SalesOrderServiceClient,
) *SalesOrderService {
	return &SalesOrderService{
		log:                     ctx.NewLoggerHelper("sales_order/service/app-service"),
		salesOrderServiceClient: salesOrderServiceClient,
	}
}

func (s *SalesOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListSalesOrderResponse, error) {
	return s.salesOrderServiceClient.List(ctx, req)
}

func (s *SalesOrderService) Get(ctx context.Context, req *salesV1.GetSalesOrderRequest) (*salesV1.SalesOrder, error) {
	return s.salesOrderServiceClient.Get(ctx, req)
}
