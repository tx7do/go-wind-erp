package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

type StockMovementService struct {
	appV1.StockMovementServiceHTTPServer

	stockMovementServiceClient inventoryV1.StockMovementServiceClient

	log *log.Helper
}

func NewStockMovementService(
	ctx *bootstrap.Context,
	stockMovementServiceClient inventoryV1.StockMovementServiceClient,
) *StockMovementService {
	return &StockMovementService{
		log:                       ctx.NewLoggerHelper("stock_movement/service/app-service"),
		stockMovementServiceClient: stockMovementServiceClient,
	}
}

func (s *StockMovementService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockMovementResponse, error) {
	return s.stockMovementServiceClient.List(ctx, req)
}

func (s *StockMovementService) Get(ctx context.Context, req *inventoryV1.GetStockMovementRequest) (*inventoryV1.StockMovement, error) {
	return s.stockMovementServiceClient.Get(ctx, req)
}

func (s *StockMovementService) Create(ctx context.Context, req *inventoryV1.CreateStockMovementRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, appV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息，created_by 由服务端从登录态推导，不信任客户端传值
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	// 主键由数据库分配，忽略客户端传入的 Id（避免自选主键撞库）。
	req.Data.Id = nil

	_, err = s.stockMovementServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}
