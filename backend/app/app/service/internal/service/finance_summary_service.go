package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// FinanceSummaryService 财务汇总（app BFF facade，移动端驾驶舱只读）。
type FinanceSummaryService struct {
	appV1.FinanceSummaryServiceHTTPServer

	financeReportServiceClient adminV1.FinanceReportServiceClient

	log *log.Helper
}

func NewFinanceSummaryService(
	ctx *bootstrap.Context,
	financeReportServiceClient adminV1.FinanceReportServiceClient,
) *FinanceSummaryService {
	return &FinanceSummaryService{
		financeReportServiceClient: financeReportServiceClient,
		log:                        ctx.NewLoggerHelper("finance_summary/service/app-service"),
	}
}

func (s *FinanceSummaryService) GetFinanceSummary(ctx context.Context, _ *emptypb.Empty) (*financeV1.FinanceSummaryResponse, error) {
	return s.financeReportServiceClient.GetFinanceSummary(ctx, &emptypb.Empty{})
}
