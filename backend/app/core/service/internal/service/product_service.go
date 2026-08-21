package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	productV1 "go-wind-erp/api/gen/go/product/service/v1"
)

// ProductService 仓库服务
type ProductService struct {
	productV1.UnimplementedProductServiceServer

	log *log.Helper

	productRepo *data.ProductRepo
}

func NewProductService(
	ctx *bootstrap.Context,
	productRepo *data.ProductRepo,
) *ProductService {
	svc := &ProductService{
		log:      ctx.NewLoggerHelper("product/service/core-service"),
		productRepo: productRepo,
	}

	return svc
}

func (s *ProductService) List(ctx context.Context, req *paginationV1.PagingRequest) (*productV1.ListProductResponse, error) {
	return s.productRepo.List(ctx, req)
}

func (s *ProductService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*productV1.CountProductResponse, error) {
	count, err := s.productRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &productV1.CountProductResponse{Count: uint64(count)}, nil
}

func (s *ProductService) Get(ctx context.Context, req *productV1.GetProductRequest) (*productV1.Product, error) {
	return s.productRepo.Get(ctx, req)
}

func (s *ProductService) Create(ctx context.Context, req *productV1.CreateProductRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.productRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ProductService) Update(ctx context.Context, req *productV1.UpdateProductRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, productV1.ErrorBadRequest("invalid parameter")
	}

	if _, err := s.productRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *ProductService) Delete(ctx context.Context, req *productV1.DeleteProductRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, productV1.ErrorBadRequest("invalid request")
	}

	if err := s.productRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
