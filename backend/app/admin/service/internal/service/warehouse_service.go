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

type WarehouseService struct {
	adminV1.WarehouseServiceHTTPServer

	log *log.Helper

	warehouseServiceClient inventoryV1.WarehouseServiceClient
}

func NewWarehouseService(
	ctx *bootstrap.Context,
	warehouseServiceClient inventoryV1.WarehouseServiceClient,
) *WarehouseService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "warehouse/service/admin-service"))
	return &WarehouseService{
		log:                   l,
		warehouseServiceClient: warehouseServiceClient,
	}
}

func (s *WarehouseService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListWarehouseResponse, error) {
	return s.warehouseServiceClient.List(ctx, req)
}

func (s *WarehouseService) Get(ctx context.Context, req *inventoryV1.GetWarehouseRequest) (*inventoryV1.Warehouse, error) {
	return s.warehouseServiceClient.Get(ctx, req)
}

func (s *WarehouseService) Create(ctx context.Context, req *inventoryV1.CreateWarehouseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.warehouseServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *WarehouseService) Update(ctx context.Context, req *inventoryV1.UpdateWarehouseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	_, err = s.warehouseServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *WarehouseService) Delete(ctx context.Context, req *inventoryV1.DeleteWarehouseRequest) (*emptypb.Empty, error) {
	return s.warehouseServiceClient.Delete(ctx, req)
}
