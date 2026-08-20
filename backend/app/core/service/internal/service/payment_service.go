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

// PaymentService 付款服务（append-only：Create/List/Get，无 Update/Delete）
type PaymentService struct {
	financeV1.UnimplementedPaymentServiceServer

	log *log.Helper

	paymentRepo *data.PaymentRepo
	payableRepo *data.PayableRepo
}

func NewPaymentService(
	ctx *bootstrap.Context,
	paymentRepo *data.PaymentRepo,
	payableRepo *data.PayableRepo,
) *PaymentService {
	svc := &PaymentService{
		log:         ctx.NewLoggerHelper("payment/service/core-service"),
		paymentRepo: paymentRepo,
		payableRepo: payableRepo,
	}

	return svc
}

func (s *PaymentService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPaymentResponse, error) {
	return s.paymentRepo.List(ctx, req)
}

func (s *PaymentService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.CountPaymentResponse, error) {
	count, err := s.paymentRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &financeV1.CountPaymentResponse{Count: uint64(count)}, nil
}

func (s *PaymentService) Get(ctx context.Context, req *financeV1.GetPaymentRequest) (*financeV1.Payment, error) {
	return s.paymentRepo.Get(ctx, req)
}

// Create 记录付款：校验金额 > 0 并做加法级上限校验（paid+payment 与总额
// 都在 int64 内），随后由 ApplyPayment 在 SQL 层原子累计并防超付竞态，
// 最后落付款流水（流水失败不影响已入账的应付——以流水为对账基准时
// 可经审计发现；反之先流水后入账会造成超付窗口，故选此序）。
func (s *PaymentService) Create(ctx context.Context, req *financeV1.CreatePaymentRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}
	if req.Data.GetPayableId() == 0 {
		return nil, financeV1.ErrorBadRequest("payable_id is required")
	}
	amount := req.Data.GetAmount()
	if amount <= 0 {
		return nil, financeV1.ErrorBadRequest("amount must be positive")
	}

	payable, err := s.payableRepo.Get(ctx, &financeV1.GetPayableRequest{
		QueryBy: &financeV1.GetPayableRequest_Id{Id: req.Data.GetPayableId()},
	})
	if err != nil {
		return nil, err
	}

	if _, overflow := addChecked(payable.GetPaidAmount(), amount); overflow {
		return nil, financeV1.ErrorBadRequest("paid amount overflow")
	}

	if _, err := s.payableRepo.ApplyPayment(ctx, req.Data.GetPayableId(), amount); err != nil {
		return nil, err
	}

	if _, err := s.paymentRepo.Create(ctx, req); err != nil {
		s.log.Errorf("insert payment ledger failed after payable applied: %s", err.Error())
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
