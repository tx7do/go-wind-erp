package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	productV1 "go-wind-erp/api/gen/go/product/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

type ProductService struct {
	adminV1.ProductServiceHTTPServer

	log *log.Helper

	productServiceClient productV1.ProductServiceClient
}

func NewProductService(
	ctx *bootstrap.Context,
	productServiceClient productV1.ProductServiceClient,
) *ProductService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "product/service/admin-service"))
	return &ProductService{
		log:                   l,
		productServiceClient: productServiceClient,
	}
}

func (s *ProductService) List(ctx context.Context, req *paginationV1.PagingRequest) (*productV1.ListProductResponse, error) {
	return s.productServiceClient.List(ctx, req)
}

func (s *ProductService) Get(ctx context.Context, req *productV1.GetProductRequest) (*productV1.Product, error) {
	return s.productServiceClient.Get(ctx, req)
}

func (s *ProductService) Create(ctx context.Context, req *productV1.CreateProductRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.productServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *ProductService) Update(ctx context.Context, req *productV1.UpdateProductRequest) (*emptypb.Empty, error) {
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

	_, err = s.productServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *ProductService) Delete(ctx context.Context, req *productV1.DeleteProductRequest) (*emptypb.Empty, error) {
	return s.productServiceClient.Delete(ctx, req)
}
