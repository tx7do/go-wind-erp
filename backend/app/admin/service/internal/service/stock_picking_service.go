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

type StockPickingService struct {
	adminV1.StockPickingServiceHTTPServer

	log *log.Helper

	stockPickingServiceClient inventoryV1.StockPickingServiceClient
}

func NewStockPickingService(
	ctx *bootstrap.Context,
	stockPickingServiceClient inventoryV1.StockPickingServiceClient,
) *StockPickingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "stock-picking/service/admin-service"))
	return &StockPickingService{
		log:                       l,
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
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.stockPickingServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
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

func (s *StockPickingService) Delete(ctx context.Context, req *inventoryV1.DeleteStockPickingRequest) (*emptypb.Empty, error) {
	return s.stockPickingServiceClient.Delete(ctx, req)
}
