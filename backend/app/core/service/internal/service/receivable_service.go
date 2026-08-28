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

// ReceivableService 应收单服务
type ReceivableService struct {
	financeV1.UnimplementedReceivableServiceServer

	log *log.Helper

	receivableRepo *data.ReceivableRepo
}

func NewReceivableService(
	ctx *bootstrap.Context,
	receivableRepo *data.ReceivableRepo,
) *ReceivableService {
	svc := &ReceivableService{
		log:            ctx.NewLoggerHelper("receivable/service/core-service"),
		receivableRepo: receivableRepo,
	}

	return svc
}

func (s *ReceivableService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceivableResponse, error) {
	return s.receivableRepo.List(ctx, req)
}

func (s *ReceivableService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.CountReceivableResponse, error) {
	count, err := s.receivableRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &financeV1.CountReceivableResponse{Count: uint64(count)}, nil
}

func (s *ReceivableService) Get(ctx context.Context, req *financeV1.GetReceivableRequest) (*financeV1.Receivable, error) {
	return s.receivableRepo.Get(ctx, req)
}

// AgingReport 应收账龄报表。
func (s *ReceivableService) AgingReport(ctx context.Context, _ *emptypb.Empty) (*financeV1.AgingReportResponse, error) {
	buckets, err := s.receivableRepo.AgingReport(ctx)
	if err != nil {
		return nil, err
	}
	return &financeV1.AgingReportResponse{Buckets: buckets}, nil
}

// Create 手工建账：校验客户与金额（正数，分）。
func (s *ReceivableService) Create(ctx context.Context, req *financeV1.CreateReceivableRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}

	if req.Data.GetCustomerCode() == "" {
		return nil, financeV1.ErrorBadRequest("customer_code is required")
	}
	if req.Data.GetAmount() <= 0 {
		return nil, financeV1.ErrorBadRequest("amount must be positive")
	}

	if _, err := s.receivableRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Delete 仅 PENDING 且未收款（条件更新守卫，审计与在途账不可抹除）。
func (s *ReceivableService) Delete(ctx context.Context, req *financeV1.DeleteReceivableRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid request")
	}

	if err := s.receivableRepo.DeleteAsUnpaid(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Cancel 仅 PENDING 且未收款（部分收款的账不可取消——先结清或调整）。
func (s *ReceivableService) Cancel(ctx context.Context, req *financeV1.CancelReceivableRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid request")
	}

	if err := s.receivableRepo.CancelAsUnpaid(ctx, req.GetId()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
