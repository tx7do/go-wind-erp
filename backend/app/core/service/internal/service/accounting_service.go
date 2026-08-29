package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-erp/app/core/service/internal/data"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// AccountingService 会计服务（简易总账，只读查询）。科目目录为平台标准
// 种子；凭证由业务事件经 journalPoster 在事务内自动生成，无手工录入口。
type AccountingService struct {
	financeV1.UnimplementedAccountingServiceServer

	log         *log.Helper
	accountRepo *data.AccountRepo
	journalRepo *data.JournalRepo
}

func NewAccountingService(
	ctx *bootstrap.Context,
	accountRepo *data.AccountRepo,
	journalRepo *data.JournalRepo,
) *AccountingService {
	// 科目种子幂等（镜像 PlanService 的目录初始化模式）。
	accountRepo.SeedIfEmpty(ctx.Context())
	return &AccountingService{
		log:         ctx.NewLoggerHelper("accounting/service/core-service"),
		accountRepo: accountRepo,
		journalRepo: journalRepo,
	}
}

func (s *AccountingService) ListAccounts(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListAccountResponse, error) {
	return s.accountRepo.List(ctx, req)
}

func (s *AccountingService) ListJournalEntries(ctx context.Context, req *financeV1.ListJournalEntryRequest) (*financeV1.ListJournalEntryResponse, error) {
	return s.journalRepo.ListJournal(ctx, req)
}

func (s *AccountingService) GetTrialBalance(ctx context.Context, req *financeV1.GetTrialBalanceRequest) (*financeV1.TrialBalanceResponse, error) {
	return s.journalRepo.TrialBalance(ctx, req)
}
