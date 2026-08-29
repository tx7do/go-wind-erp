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

// AgingReport 应付账龄报表（委派 core）。
func (s *PayableService) AgingReport(ctx context.Context, req *emptypb.Empty) (*financeV1.AgingReportResponse, error) {
	return s.payableServiceClient.AgingReport(ctx, req)
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

// ReceivableService 应收单管理（admin BFF facade）
type ReceivableService struct {
	adminV1.ReceivableServiceHTTPServer

	log *log.Helper

	receivableServiceClient financeV1.ReceivableServiceClient
}

func NewReceivableService(
	ctx *bootstrap.Context,
	receivableServiceClient financeV1.ReceivableServiceClient,
) *ReceivableService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "receivable/service/admin-service"))
	return &ReceivableService{
		log:                     l,
		receivableServiceClient: receivableServiceClient,
	}
}

func (s *ReceivableService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceivableResponse, error) {
	return s.receivableServiceClient.List(ctx, req)
}

func (s *ReceivableService) Get(ctx context.Context, req *financeV1.GetReceivableRequest) (*financeV1.Receivable, error) {
	return s.receivableServiceClient.Get(ctx, req)
}

// AgingReport 应收账龄报表（委派 core）。
func (s *ReceivableService) AgingReport(ctx context.Context, req *emptypb.Empty) (*financeV1.AgingReportResponse, error) {
	return s.receivableServiceClient.AgingReport(ctx, req)
}

func (s *ReceivableService) Create(ctx context.Context, req *financeV1.CreateReceivableRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.receivableServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *ReceivableService) Delete(ctx context.Context, req *financeV1.DeleteReceivableRequest) (*emptypb.Empty, error) {
	return s.receivableServiceClient.Delete(ctx, req)
}

func (s *ReceivableService) Cancel(ctx context.Context, req *financeV1.CancelReceivableRequest) (*emptypb.Empty, error) {
	return s.receivableServiceClient.Cancel(ctx, req)
}

// ReceiptService 收款管理（admin BFF facade，append-only）
type ReceiptService struct {
	adminV1.ReceiptServiceHTTPServer

	log *log.Helper

	receiptServiceClient financeV1.ReceiptServiceClient
}

func NewReceiptService(
	ctx *bootstrap.Context,
	receiptServiceClient financeV1.ReceiptServiceClient,
) *ReceiptService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "receipt/service/admin-service"))
	return &ReceiptService{
		log:                  l,
		receiptServiceClient: receiptServiceClient,
	}
}

func (s *ReceiptService) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListReceiptResponse, error) {
	return s.receiptServiceClient.List(ctx, req)
}

func (s *ReceiptService) Get(ctx context.Context, req *financeV1.GetReceiptRequest) (*financeV1.Receipt, error) {
	return s.receiptServiceClient.Get(ctx, req)
}

func (s *ReceiptService) Create(ctx context.Context, req *financeV1.CreateReceiptRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.receiptServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

// FinanceReportService 财务报表（admin BFF facade）
type FinanceReportService struct {
	adminV1.FinanceReportServiceHTTPServer

	log *log.Helper

	financeReportServiceClient adminV1.FinanceReportServiceClient
}

func NewFinanceReportService(
	ctx *bootstrap.Context,
	financeReportServiceClient adminV1.FinanceReportServiceClient,
) *FinanceReportService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "finance_report/service/admin-service"))
	return &FinanceReportService{
		log:                       l,
		financeReportServiceClient: financeReportServiceClient,
	}
}

// ProfitReport 利润月度报表：收入（已完成销售单）− COGS（出库冻结成本）。
func (s *FinanceReportService) ProfitReport(ctx context.Context, req *emptypb.Empty) (*financeV1.ProfitReportResponse, error) {
	return s.financeReportServiceClient.ProfitReport(ctx, req)
}

// GetFinanceSummary 经营汇总（驾驶舱）。
func (s *FinanceReportService) GetFinanceSummary(ctx context.Context, req *emptypb.Empty) (*financeV1.FinanceSummaryResponse, error) {
	return s.financeReportServiceClient.GetFinanceSummary(ctx, req)
}

// GetSalesRanking 销售排行（按商品/按客户）。
func (s *FinanceReportService) GetSalesRanking(ctx context.Context, req *financeV1.GetSalesRankingRequest) (*financeV1.SalesRankingResponse, error) {
	return s.financeReportServiceClient.GetSalesRanking(ctx, req)
}

// GetPartnerStatement 往来对账单（客户/供应商）。
func (s *FinanceReportService) GetPartnerStatement(ctx context.Context, req *financeV1.GetPartnerStatementRequest) (*financeV1.PartnerStatementResponse, error) {
	return s.financeReportServiceClient.GetPartnerStatement(ctx, req)
}
