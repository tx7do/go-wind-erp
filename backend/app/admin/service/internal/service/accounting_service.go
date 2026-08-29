package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// AccountingService 会计（admin BFF facade，纯委派 core）。
type AccountingService struct {
	adminV1.AccountingServiceHTTPServer

	log *log.Helper

	accountingServiceClient financeV1.AccountingServiceClient
}

func NewAccountingService(
	ctx *bootstrap.Context,
	accountingServiceClient financeV1.AccountingServiceClient,
) *AccountingService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "accounting/service/admin-service"))
	return &AccountingService{
		log:                     l,
		accountingServiceClient: accountingServiceClient,
	}
}

func (s *AccountingService) ListAccounts(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListAccountResponse, error) {
	return s.accountingServiceClient.ListAccounts(ctx, req)
}

func (s *AccountingService) ListJournalEntries(ctx context.Context, req *financeV1.ListJournalEntryRequest) (*financeV1.ListJournalEntryResponse, error) {
	return s.accountingServiceClient.ListJournalEntries(ctx, req)
}

func (s *AccountingService) GetTrialBalance(ctx context.Context, req *financeV1.GetTrialBalanceRequest) (*financeV1.TrialBalanceResponse, error) {
	return s.accountingServiceClient.GetTrialBalance(ctx, req)
}
