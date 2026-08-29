package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	"github.com/tx7do/go-utils/trans"
	"go-wind-erp/pkg/constants"

	adminpb "go-wind-erp/api/gen/go/admin/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// FinanceReportService 财务报表服务
type FinanceReportService struct {
	adminpb.UnimplementedFinanceReportServiceServer

	log *log.Helper

	salesOrderRepo    *data.SalesOrderRepo
	stockPickingRepo  *data.StockPickingRepo
	stockMoveLineRepo *data.StockMoveLineRepo
	journalRepo       *data.JournalRepo
	receivableRepo    *data.ReceivableRepo
	payableRepo       *data.PayableRepo
}

func NewFinanceReportService(
	ctx *bootstrap.Context,
	salesOrderRepo *data.SalesOrderRepo,
	stockPickingRepo *data.StockPickingRepo,
	stockMoveLineRepo *data.StockMoveLineRepo,
	journalRepo *data.JournalRepo,
	receivableRepo *data.ReceivableRepo,
	payableRepo *data.PayableRepo,
) *FinanceReportService {
	svc := &FinanceReportService{
		log:               ctx.NewLoggerHelper("finance_report/service/core-service"),
		salesOrderRepo:    salesOrderRepo,
		stockPickingRepo:  stockPickingRepo,
		stockMoveLineRepo: stockMoveLineRepo,
		journalRepo:       journalRepo,
		receivableRepo:    receivableRepo,
		payableRepo:       payableRepo,
	}
	return svc
}

// ProfitReport 利润报表：按月汇总收入（COMPLETED 销售单 total_amount 之和）
// 与 COGS（出库 move-line 的 executed_quantity × unit_cost 之和），利润=收入−成本。
func (s *FinanceReportService) ProfitReport(ctx context.Context, _ *emptypb.Empty) (*financeV1.ProfitReportResponse, error) {
	// 取近 12 个月的数据
	now := time.Now()
	months := make([]string, 12)
	for i := 11; i >= 0; i-- {
		months[11-i] = now.AddDate(0, -i, 0).Format("2006-01")
	}

	// 收入：按月汇总 COMPLETED 销售单的 total_amount
	revenueRows, err := s.salesOrderRepo.RevenueByMonth(ctx)
	if err != nil {
		return nil, err
	}
	revenueMap := make(map[string]int64)
	for _, row := range revenueRows {
		if row.Month == "" {
			continue
		}
		revenueMap[row.Month] = row.Total
	}

	// COGS：取 OUTGOING 拣货单 ID，再按月汇总其 move-line 的
	// executed_quantity × unit_cost
	outgoingIDs, err := s.stockPickingRepo.GetOutgoingPickingIDs(ctx)
	if err != nil {
		return nil, err
	}
	cogsMap := make(map[string]int64)
	if len(outgoingIDs) > 0 {
		cogsRows, err := s.stockMoveLineRepo.CogsByMonth(ctx, outgoingIDs)
		if err != nil {
			return nil, err
		}
		for _, row := range cogsRows {
			if row.Month == "" {
				continue
			}
			cogsMap[row.Month] = row.Total
		}
	}

	// 组装：12 个月的固定列表，缺失月份补 0
	items := make([]*financeV1.MonthlyProfit, 0, 12)
	for _, m := range months {
		mCopy := m
		rev := revenueMap[m]
		cost := cogsMap[m]
		profit := rev - cost
		items = append(items, &financeV1.MonthlyProfit{
			Month:   &mCopy,
			Revenue: &rev,
			Cogs:    &cost,
			Profit:  &profit,
		})
	}

	return &financeV1.ProfitReportResponse{Items: items}, nil
}



// GetFinanceSummary 经营汇总（驾驶舱）：本月收入/成本取总账 6001/6401
// 当月净额（冲回自动抵减），利润=收入−成本；应收/应付未清余额来自业务台账。
func (s *FinanceReportService) GetFinanceSummary(ctx context.Context, _ *emptypb.Empty) (*financeV1.FinanceSummaryResponse, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	revDebit, revCredit, err := s.journalRepo.AccountNetInRange(ctx, constants.AccountCodeRevenue, monthStart, now)
	if err != nil {
		return nil, err
	}
	cogsDebit, cogsCredit, err := s.journalRepo.AccountNetInRange(ctx, constants.AccountCodeCOGS, monthStart, now)
	if err != nil {
		return nil, err
	}
	arBalance, err := s.receivableRepo.OutstandingBalance(ctx)
	if err != nil {
		return nil, err
	}
	apBalance, err := s.payableRepo.OutstandingBalance(ctx)
	if err != nil {
		return nil, err
	}

	revenue := revCredit - revDebit
	cogs := cogsDebit - cogsCredit
	return &financeV1.FinanceSummaryResponse{
		RevenueMonth: &revenue,
		CogsMonth:    &cogs,
		ProfitMonth:  trans.Ptr(revenue - cogs),
		ArBalance:    &arBalance,
		ApBalance:    &apBalance,
	}, nil
}
