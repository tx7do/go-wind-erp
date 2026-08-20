package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

type StockMovementService struct {
	adminV1.StockMovementServiceHTTPServer

	log *log.Helper

	stockMovementServiceClient inventoryV1.StockMovementServiceClient
}

func NewStockMovementService(
	ctx *bootstrap.Context,
	stockMovementServiceClient inventoryV1.StockMovementServiceClient,
) *StockMovementService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "stock_movement/service/admin-service"))
	return &StockMovementService{
		log:                       l,
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
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.stockMovementServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *StockMovementService) Delete(ctx context.Context, req *inventoryV1.DeleteStockMovementRequest) (*emptypb.Empty, error) {
	return s.stockMovementServiceClient.Delete(ctx, req)
}
