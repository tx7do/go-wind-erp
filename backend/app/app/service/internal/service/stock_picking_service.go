package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

type StockPickingService struct {
	appV1.StockPickingServiceHTTPServer

	stockPickingServiceClient inventoryV1.StockPickingServiceClient

	log *log.Helper
}

func NewStockPickingService(
	ctx *bootstrap.Context,
	stockPickingServiceClient inventoryV1.StockPickingServiceClient,
) *StockPickingService {
	return &StockPickingService{
		log:                       ctx.NewLoggerHelper("stock_picking/service/app-service"),
		stockPickingServiceClient: stockPickingServiceClient,
	}
}

func (s *StockPickingService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockPickingResponse, error) {
	return s.stockPickingServiceClient.List(ctx, req)
}

func (s *StockPickingService) Get(ctx context.Context, req *inventoryV1.GetStockPickingRequest) (*inventoryV1.StockPicking, error) {
	return s.stockPickingServiceClient.Get(ctx, req)
}

func (s *StockPickingService) Create(ctx context.Context, req *inventoryV1.CreateStockPickingRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.Create(ctx, req)
}

func (s *StockPickingService) Confirm(ctx context.Context, req *inventoryV1.ConfirmStockPickingRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.Confirm(ctx, req)
}

func (s *StockPickingService) Validate(ctx context.Context, req *inventoryV1.ValidateStockPickingRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.Validate(ctx, req)
}

func (s *StockPickingService) Cancel(ctx context.Context, req *inventoryV1.CancelStockPickingRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.Cancel(ctx, req)
}

// CreateSalesReturn 创建销售退货拣货单（纯委派，数量守卫在 core）。
func (s *StockPickingService) CreateSalesReturn(ctx context.Context, req *inventoryV1.CreateSalesReturnRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.CreateSalesReturn(ctx, req)
}

// CreatePurchaseReturn 创建采购退货拣货单（纯委派，数量守卫在 core）。
func (s *StockPickingService) CreatePurchaseReturn(ctx context.Context, req *inventoryV1.CreatePurchaseReturnRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.CreatePurchaseReturn(ctx, req)
}
