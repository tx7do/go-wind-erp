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

type InventoryService struct {
	adminV1.InventoryServiceHTTPServer

	log *log.Helper

	inventoryServiceClient inventoryV1.InventoryServiceClient
}

func NewInventoryService(
	ctx *bootstrap.Context,
	inventoryServiceClient inventoryV1.InventoryServiceClient,
) *InventoryService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "inventory/service/admin-service"))
	return &InventoryService{
		log:                   l,
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

func (s *InventoryService) Create(ctx context.Context, req *inventoryV1.CreateInventoryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.inventoryServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *InventoryService) Update(ctx context.Context, req *inventoryV1.UpdateInventoryRequest) (*emptypb.Empty, error) {
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

	_, err = s.inventoryServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *InventoryService) Delete(ctx context.Context, req *inventoryV1.DeleteInventoryRequest) (*emptypb.Empty, error) {
	return s.inventoryServiceClient.Delete(ctx, req)
}
