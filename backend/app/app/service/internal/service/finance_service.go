package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// PayableService 应付单（app BFF facade，移动端只读）。
type PayableService struct {
	appV1.PayableServiceHTTPServer

	payableServiceClient financeV1.PayableServiceClient

	log *log.Helper
}

func NewPayableService(
	ctx *bootstrap.Context,
	payableServiceClient financeV1.PayableServiceClient,
) *PayableService {
	return &PayableService{
		log:                  ctx.NewLoggerHelper("payable/service/app-service"),
		payableServiceClient: payableServiceClient,
	}
}

func (s *PayableService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPayableResponse, error) {
	return s.payableServiceClient.List(ctx, req)
}

func (s *PayableService) Get(ctx context.Context, req *financeV1.GetPayableRequest) (*financeV1.Payable, error) {
	return s.payableServiceClient.Get(ctx, req)
}

// PaymentService 付款（app BFF facade，移动端只读台账）。
type PaymentService struct {
	appV1.PaymentServiceHTTPServer

	paymentServiceClient financeV1.PaymentServiceClient

	log *log.Helper
}

func NewPaymentService(
	ctx *bootstrap.Context,
	paymentServiceClient financeV1.PaymentServiceClient,
) *PaymentService {
	return &PaymentService{
		log:                  ctx.NewLoggerHelper("payment/service/app-service"),
		paymentServiceClient: paymentServiceClient,
	}
}

func (s *PaymentService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPaymentResponse, error) {
	return s.paymentServiceClient.List(ctx, req)
}

func (s *PaymentService) Get(ctx context.Context, req *financeV1.GetPaymentRequest) (*financeV1.Payment, error) {
	return s.paymentServiceClient.Get(ctx, req)
}
