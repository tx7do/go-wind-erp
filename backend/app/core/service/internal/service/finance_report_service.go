package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	"fmt"
	"sort"

	"github.com/tx7do/go-utils/trans"
	"go-wind-erp/pkg/constants"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	receiptRepo       *data.ReceiptRepo
	paymentRepo       *data.PaymentRepo
}

func NewFinanceReportService(
	ctx *bootstrap.Context,
	salesOrderRepo *data.SalesOrderRepo,
	stockPickingRepo *data.StockPickingRepo,
	stockMoveLineRepo *data.StockMoveLineRepo,
	journalRepo *data.JournalRepo,
	receivableRepo *data.ReceivableRepo,
	payableRepo *data.PayableRepo,
	receiptRepo *data.ReceiptRepo,
	paymentRepo *data.PaymentRepo,
) *FinanceReportService {
	svc := &FinanceReportService{
		log:               ctx.NewLoggerHelper("finance_report/service/core-service"),
		salesOrderRepo:    salesOrderRepo,
		stockPickingRepo:  stockPickingRepo,
		stockMoveLineRepo: stockMoveLineRepo,
		journalRepo:       journalRepo,
		receivableRepo:    receivableRepo,
		payableRepo:       payableRepo,
		receiptRepo:       receiptRepo,
		paymentRepo:       paymentRepo,
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


// GetPartnerStatement 往来对账单：客户 = 应收发生 + 收款核销；
// 供应商 = 应付发生 + 付款核销。按日期升序，期末余额 = 发生 − 核销。
// 收付款以 APPLIED 为准（PENDING 未入账、REJECTED 无效）。
func (s *FinanceReportService) GetPartnerStatement(
	ctx context.Context,
	req *financeV1.GetPartnerStatementRequest,
) (*financeV1.PartnerStatementResponse, error) {
	if req == nil || req.GetPartnerCode() == "" {
		return nil, financeV1.ErrorBadRequest("partner_code is required")
	}
	switch req.GetPartnerType() {
	case "CUSTOMER":
		return s.customerStatement(ctx, req)
	case "SUPPLIER":
		return s.supplierStatement(ctx, req)
	default:
		return nil, financeV1.ErrorBadRequest("partner_type must be CUSTOMER or SUPPLIER")
	}
}

func inRange(t *timestamppb.Timestamp, from, to *timestamppb.Timestamp) bool {
	if t == nil {
		return false
	}
	if from != nil && t.AsTime().Before(from.AsTime()) {
		return false
	}
	if to != nil && t.AsTime().After(to.AsTime()) {
		return false
	}
	return true
}

func (s *FinanceReportService) customerStatement(
	ctx context.Context,
	req *financeV1.GetPartnerStatementRequest,
) (*financeV1.PartnerStatementResponse, error) {
	resp := &financeV1.PartnerStatementResponse{
		PartnerType: req.PartnerType,
		PartnerCode: req.PartnerCode,
	}

	// 应收发生（含退货冲抵后的当前净额——与总账口径一致）。
	recs, err := s.receivableRepo.ListByCustomer(ctx, req.GetPartnerCode())
	if err != nil {
		return nil, err
	}
	recByID := make(map[uint32]*financeV1.Receivable, len(recs))
	for _, rec := range recs {
		recByID[rec.GetId()] = rec
		if inRange(rec.CreatedAt, req.FromDate, req.ToDate) {
			resp.Rows = append(resp.Rows, &financeV1.PartnerStatementRow{
				Date:    rec.CreatedAt,
				DocType: trans.Ptr("销售应收"),
				DocRef:  trans.Ptr(rec.GetSoRef()),
				Summary: trans.Ptr(fmt.Sprintf("应收 %s", centsLabel(rec.GetAmount()))),
				Debit:   trans.Ptr(rec.GetAmount()),
			})
		}
	}

	// 收款核销（仅 APPLIED）。
	receipts, err := s.receiptRepo.ListApplied(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range receipts {
		rec := recByID[r.GetReceivableId()]
		if rec == nil {
			continue
		}
		if !inRange(r.CreatedAt, req.FromDate, req.ToDate) {
			continue
		}
		resp.Rows = append(resp.Rows, &financeV1.PartnerStatementRow{
			Date:    r.CreatedAt,
			DocType: trans.Ptr("收款"),
			DocRef:  trans.Ptr(rec.GetSoRef()),
			Summary: trans.Ptr(fmt.Sprintf("收款 %s（%s）", centsLabel(r.GetAmount()), receiptMethodLabel(int32(r.GetMethod())))),
			Credit:  trans.Ptr(r.GetAmount()),
		})
	}

	finalizeStatement(resp)
	return resp, nil
}

func (s *FinanceReportService) supplierStatement(
	ctx context.Context,
	req *financeV1.GetPartnerStatementRequest,
) (*financeV1.PartnerStatementResponse, error) {
	resp := &financeV1.PartnerStatementResponse{
		PartnerType: req.PartnerType,
		PartnerCode: req.PartnerCode,
	}

	pays, err := s.payableRepo.ListBySupplier(ctx, req.GetPartnerCode())
	if err != nil {
		return nil, err
	}
	payByID := make(map[uint32]*financeV1.Payable, len(pays))
	for _, p := range pays {
		payByID[p.GetId()] = p
		if inRange(p.CreatedAt, req.FromDate, req.ToDate) {
			resp.Rows = append(resp.Rows, &financeV1.PartnerStatementRow{
				Date:    p.CreatedAt,
				DocType: trans.Ptr("采购应付"),
				DocRef:  trans.Ptr(p.GetPoRef()),
				Summary: trans.Ptr(fmt.Sprintf("应付 %s", centsLabel(p.GetAmount()))),
				Debit:   trans.Ptr(p.GetAmount()),
			})
		}
	}

	payments, err := s.paymentRepo.ListApplied(ctx)
	if err != nil {
		return nil, err
	}
	for _, pm := range payments {
		pay := payByID[pm.GetPayableId()]
		if pay == nil {
			continue
		}
		if !inRange(pm.CreatedAt, req.FromDate, req.ToDate) {
			continue
		}
		resp.Rows = append(resp.Rows, &financeV1.PartnerStatementRow{
			Date:    pm.CreatedAt,
			DocType: trans.Ptr("付款"),
			DocRef:  trans.Ptr(pay.GetPoRef()),
			Summary: trans.Ptr(fmt.Sprintf("付款 %s（%s）", centsLabel(pm.GetAmount()), receiptMethodLabel(int32(pm.GetMethod())))),
			Credit:  trans.Ptr(pm.GetAmount()),
		})
	}

	finalizeStatement(resp)
	return resp, nil
}

// finalizeStatement 按日期升序排列并合计。
func finalizeStatement(resp *financeV1.PartnerStatementResponse) {
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		return resp.Rows[i].GetDate().AsTime().Before(resp.Rows[j].GetDate().AsTime())
	})
	var totalDebit, totalCredit int64
	for _, row := range resp.Rows {
		totalDebit += row.GetDebit()
		totalCredit += row.GetCredit()
	}
	resp.TotalDebit, resp.TotalCredit = &totalDebit, &totalCredit
	balance := totalDebit - totalCredit
	resp.Balance = &balance
}

func centsLabel(cents int64) string {
	return fmt.Sprintf("%.2f 元", float64(cents)/100)
}

func receiptMethodLabel(m int32) string {
	switch m {
	case 1:
		return "现金"
	case 2:
		return "支票"
	case 0:
		return "银行转账"
	default:
		return "其他"
	}
}
