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

type LocationService struct {
	adminV1.LocationServiceHTTPServer

	log *log.Helper

	locationServiceClient inventoryV1.LocationServiceClient
}

func NewLocationService(
	ctx *bootstrap.Context,
	locationServiceClient inventoryV1.LocationServiceClient,
) *LocationService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "location/service/admin-service"))
	return &LocationService{
		log:                   l,
		locationServiceClient: locationServiceClient,
	}
}

func (s *LocationService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListLocationResponse, error) {
	return s.locationServiceClient.List(ctx, req)
}

func (s *LocationService) Get(ctx context.Context, req *inventoryV1.GetLocationRequest) (*inventoryV1.StockLocation, error) {
	return s.locationServiceClient.Get(ctx, req)
}

func (s *LocationService) Create(ctx context.Context, req *inventoryV1.CreateLocationRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.locationServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *LocationService) Update(ctx context.Context, req *inventoryV1.UpdateLocationRequest) (*emptypb.Empty, error) {
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

	_, err = s.locationServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *LocationService) Delete(ctx context.Context, req *inventoryV1.DeleteLocationRequest) (*emptypb.Empty, error) {
	return s.locationServiceClient.Delete(ctx, req)
}
