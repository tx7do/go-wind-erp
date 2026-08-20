package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// PayableService 应付单服务
type PayableService struct {
	financeV1.UnimplementedPayableServiceServer

	log *log.Helper

	payableRepo *data.PayableRepo
}

func NewPayableService(
	ctx *bootstrap.Context,
	payableRepo *data.PayableRepo,
) *PayableService {
	svc := &PayableService{
		log:        ctx.NewLoggerHelper("payable/service/core-service"),
		payableRepo: payableRepo,
	}

	return svc
}

func (s *PayableService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPayableResponse, error) {
	return s.payableRepo.List(ctx, req)
}

func (s *PayableService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.CountPayableResponse, error) {
	count, err := s.payableRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &financeV1.CountPayableResponse{Count: uint64(count)}, nil
}

func (s *PayableService) Get(ctx context.Context, req *financeV1.GetPayableRequest) (*financeV1.Payable, error) {
	return s.payableRepo.Get(ctx, req)
}

// Create 手工建账：校验供应商与金额（正数，分）。
func (s *PayableService) Create(ctx context.Context, req *financeV1.CreatePayableRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	if req.Data.GetSupplierCode() == "" {
		return nil, financeV1.ErrorBadRequest("supplier_code is required")
	}
	if req.Data.GetAmount() <= 0 {
		return nil, financeV1.ErrorBadRequest("amount must be positive")
	}

	if _, err := s.payableRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Delete 仅 PENDING 且未付款（条件更新守卫，审计与在途账不可抹除）。
func (s *PayableService) Delete(ctx context.Context, req *financeV1.DeletePayableRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid request")
	}

	if err := s.payableRepo.DeleteAsUnpaid(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Cancel 仅 PENDING 且未付款（部分付款的账不可取消——先结清或调整）。
func (s *PayableService) Cancel(ctx context.Context, req *financeV1.CancelPayableRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid request")
	}

	if err := s.payableRepo.CancelAsUnpaid(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
