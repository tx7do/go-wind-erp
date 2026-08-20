package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

// PayableService 应付单管理（admin BFF facade）
type PayableService struct {
	adminV1.PayableServiceHTTPServer

	log *log.Helper

	payableServiceClient financeV1.PayableServiceClient
}

func NewPayableService(
	ctx *bootstrap.Context,
	payableServiceClient financeV1.PayableServiceClient,
) *PayableService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "payable/service/admin-service"))
	return &PayableService{
		log:                  l,
		payableServiceClient: payableServiceClient,
	}
}

func (s *PayableService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPayableResponse, error) {
	return s.payableServiceClient.List(ctx, req)
}

func (s *PayableService) Get(ctx context.Context, req *financeV1.GetPayableRequest) (*financeV1.Payable, error) {
	return s.payableServiceClient.Get(ctx, req)
}

func (s *PayableService) Create(ctx context.Context, req *financeV1.CreatePayableRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.payableServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *PayableService) Delete(ctx context.Context, req *financeV1.DeletePayableRequest) (*emptypb.Empty, error) {
	return s.payableServiceClient.Delete(ctx, req)
}

func (s *PayableService) Cancel(ctx context.Context, req *financeV1.CancelPayableRequest) (*emptypb.Empty, error) {
	return s.payableServiceClient.Cancel(ctx, req)
}

// PaymentService 付款管理（admin BFF facade，append-only）
type PaymentService struct {
	adminV1.PaymentServiceHTTPServer

	log *log.Helper

	paymentServiceClient financeV1.PaymentServiceClient
}

func NewPaymentService(
	ctx *bootstrap.Context,
	paymentServiceClient financeV1.PaymentServiceClient,
) *PaymentService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "payment/service/admin-service"))
	return &PaymentService{
		log:                  l,
		paymentServiceClient: paymentServiceClient,
	}
}

func (s *PaymentService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListPaymentResponse, error) {
	return s.paymentServiceClient.List(ctx, req)
}

func (s *PaymentService) Get(ctx context.Context, req *financeV1.GetPaymentRequest) (*financeV1.Payment, error) {
	return s.paymentServiceClient.Get(ctx, req)
}

func (s *PaymentService) Create(ctx context.Context, req *financeV1.CreatePaymentRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.paymentServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}
