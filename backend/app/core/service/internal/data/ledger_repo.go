package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/account"
	"go-wind-erp/app/core/service/internal/data/ent/journalentry"
	"go-wind-erp/app/core/service/internal/data/ent/journalline"

	"go-wind-erp/pkg/constants"

	appViewer "go-wind-erp/pkg/entgo/viewer"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// AccountRepo 会计科目仓储。平台标准目录（tenant_id=0）：种子幂等，
// 查询一律系统视图（镜像 Plan 目录语义）。
type AccountRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

func NewAccountRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *AccountRepo {
	return &AccountRepo{
		log:       ctx.NewLoggerHelper("account/repo/core-service"),
		entClient: entClient,
	}
}

// SeedIfEmpty 幂等种子标准科目（空表才写，已存在不动）。
func (r *AccountRepo) SeedIfEmpty(ctx context.Context) {
	sysCtx := appViewer.NewSystemViewerContext(ctx)
	if n := r.entClient.Client().Account.Query().CountX(sysCtx); n > 0 {
		return
	}
	for _, a := range constants.DefaultAccounts {
		if _, err := r.entClient.Client().Account.Create().
			SetTenantID(0).
			SetCode(a.Code).
			SetName(a.Name).
			SetCategory(account.Category(a.Category)).
			SetBalanceDirection(account.BalanceDirection(a.BalanceDirection)).
			SetCreatedAt(time.Now()).
			Save(sysCtx); err != nil {
			r.log.Errorf("seed account %s failed: %s", a.Code, err.Error())
			return
		}
	}
	r.log.Infof("seeded %d default accounts", len(constants.DefaultAccounts))
}

// List 科目目录（系统视图）。
func (r *AccountRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*financeV1.ListAccountResponse, error) {
	rows, err := r.entClient.Client().Account.Query().
		Order(ent.Asc(account.FieldCode)).
		All(appViewer.NewSystemViewerContext(ctx))
	if err != nil {
		r.log.Errorf("list accounts failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("list accounts failed")
	}
	items := make([]*financeV1.Account, 0, len(rows))
	for _, a := range rows {
		items = append(items, &financeV1.Account{
			Id:               &a.ID,
			Code:             a.Code,
			Name:             a.Name,
			Category:         categoryToString(a.Category),
			BalanceDirection: directionToString(a.BalanceDirection),
		})
	}
	return &financeV1.ListAccountResponse{Total: uint64(len(items)), Items: items}, nil
}

func categoryToString(c *account.Category) *string {
	if c == nil {
		return nil
	}
	s := string(*c)
	return &s
}

func directionToString(d *account.BalanceDirection) *string {
	if d == nil {
		return nil
	}
	s := string(*d)
	return &s
}

// AllAccounts 全量科目（编码→DTO，余额表拼装用；系统视图）。
func (r *AccountRepo) AllAccounts(ctx context.Context) (map[string]*financeV1.Account, error) {
	resp, err := r.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	m := make(map[string]*financeV1.Account, len(resp.GetItems()))
	for _, a := range resp.GetItems() {
		m[a.GetCode()] = a
	}
	return m, nil
}

// JournalLineInput 过账入参行。
type JournalLineInput struct {
	AccountCode string
	Summary     string
	Debit       int64
	Credit      int64
}

// JournalRepo 记账凭证仓储。PostTx 强制：科目必须存在 + 借贷合计相等。
// append-only；凭证号 JE-<毫秒时间戳>。
type JournalRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	accountRepo *AccountRepo
}

func NewJournalRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
	accountRepo *AccountRepo,
) *JournalRepo {
	return &JournalRepo{
		log:         ctx.NewLoggerHelper("journal/repo/core-service"),
		entClient:   entClient,
		accountRepo: accountRepo,
	}
}

// PostTx 事务内过账：校验科目与平衡后写凭证头 + 行。任一失败回滚由
// 调用方事务兜底（过账与业务事件原子——账实一致优先）。
func (r *JournalRepo) PostTx(
	ctx context.Context,
	tx *ent.Tx,
	summary string,
	bizRef string,
	lines []JournalLineInput,
) error {
	if len(lines) < 2 {
		return financeV1.ErrorBadRequest("journal entry requires at least 2 lines")
	}

	var debit, credit int64
	validAccounts, err := r.accountRepo.AllAccounts(ctx)
	if err != nil {
		return err
	}
	for _, l := range lines {
		if l.Debit < 0 || l.Credit < 0 || (l.Debit > 0 && l.Credit > 0) {
			return financeV1.ErrorBadRequest("journal line amount invalid: %s", l.AccountCode)
		}
		if _, ok := validAccounts[l.AccountCode]; !ok {
			return financeV1.ErrorBadRequest("unknown account code: %s", l.AccountCode)
		}
		debit += l.Debit
		credit += l.Credit
	}
	if debit != credit {
		return financeV1.ErrorBadRequest(
			"journal entry unbalanced: debit %d != credit %d", debit, credit)
	}

	entry, err := tx.JournalEntry.Create().
		SetTenantID(mustTenantID(ctx)).
		SetEntryNumber(fmt.Sprintf("JE-%d", time.Now().UnixMilli())).
		SetSummary(summary).
		SetBizRef(bizRef).
		SetEntryDate(time.Now()).
		SetCreatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		r.log.Errorf("insert journal entry failed: %s", err.Error())
		return financeV1.ErrorInternalServerError("insert journal entry failed")
	}

	for _, l := range lines {
		if _, err := tx.JournalLine.Create().
			SetTenantID(mustTenantID(ctx)).
			SetEntryID(entry.ID).
			SetAccountCode(l.AccountCode).
			SetSummary(l.Summary).
			SetDebit(l.Debit).
			SetCredit(l.Credit).
			SetCreatedAt(time.Now()).
			Save(ctx); err != nil {
			r.log.Errorf("insert journal line failed: %s", err.Error())
			return financeV1.ErrorInternalServerError("insert journal line failed")
		}
	}
	return nil
}

// Post 非事务过账（审批联动等无外层事务场景）。
func (r *JournalRepo) Post(
	ctx context.Context,
	summary string,
	bizRef string,
	lines []JournalLineInput,
) error {
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return financeV1.ErrorInternalServerError("start transaction failed")
	}
	if err := r.PostTx(ctx, tx, summary, bizRef, lines); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return financeV1.ErrorInternalServerError("commit journal failed")
	}
	return nil
}

func mustTenantID(ctx context.Context) uint32 {
	tid, _ := maybeTenantFromViewer(ctx)
	return tid
}

// ListJournal 凭证流水（含行；日期/科目过滤，Go 侧分页——凭证量有限）。
func (r *JournalRepo) ListJournal(
	ctx context.Context,
	req *financeV1.ListJournalEntryRequest,
) (*financeV1.ListJournalEntryResponse, error) {
	if req == nil {
		return nil, financeV1.ErrorBadRequest("invalid parameter")
	}
	tid, _ := maybeTenantFromViewer(ctx)

	q := r.entClient.Client().JournalEntry.Query().
		Where(journalentry.TenantIDEQ(tid))

	// 科目过滤：先查命中行所属凭证 ID 集合再过滤头。
	if code := req.GetAccountCode(); code != "" {
		entryIDs, err := r.entClient.Client().JournalLine.Query().
			Where(
				journalline.TenantIDEQ(tid),
				journalline.AccountCodeEQ(code),
			).
			IDs(ctx)
		if err != nil || len(entryIDs) == 0 {
			return &financeV1.ListJournalEntryResponse{Total: 0, Items: []*financeV1.JournalEntry{}}, nil
		}
		q = q.Where(journalentry.IDIn(entryIDs...))
	}

	entries, err := q.Order(ent.Desc(journalentry.FieldID)).All(ctx)
	if err != nil {
		r.log.Errorf("list journal entries failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("list journal entries failed")
	}

	// 日期过滤在 Go 侧（entry_date 可空，空视为创建即入账）。
	var filtered []*ent.JournalEntry
	for _, e := range entries {
		d := e.EntryDate
		if d == nil {
			continue
		}
		if req.GetFromDate() != nil && d.Before(req.GetFromDate().AsTime()) {
			continue
		}
		if req.GetToDate() != nil && d.After(req.GetToDate().AsTime()) {
			continue
		}
		filtered = append(filtered, e)
	}

	entryIDs := make([]uint32, 0, len(filtered))
	for _, e := range filtered {
		entryIDs = append(entryIDs, e.ID)
	}
	lineRows, err := r.entClient.Client().JournalLine.Query().
		Where(journalline.EntryIDIn(entryIDs...)).
		Order(ent.Asc(journalline.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list journal lines failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("list journal lines failed")
	}
	linesByEntry := map[uint32][]*financeV1.JournalLine{}
	for _, l := range lineRows {
		if l.EntryID == nil {
			continue
		}
		linesByEntry[*l.EntryID] = append(linesByEntry[*l.EntryID], &financeV1.JournalLine{
			Id:          &l.ID,
			AccountCode: l.AccountCode,
			Summary:     l.Summary,
			Debit:       l.Debit,
			Credit:      l.Credit,
		})
	}

	items := make([]*financeV1.JournalEntry, 0, len(filtered))
	for _, e := range filtered {
		item := &financeV1.JournalEntry{
			Id:          &e.ID,
			EntryNumber: e.EntryNumber,
			Summary:     e.Summary,
			BizRef:      e.BizRef,
			Lines:       linesByEntry[e.ID],
		}
		if e.EntryDate != nil {
			item.EntryDate = timestamppb.New(*e.EntryDate)
		}
		items = append(items, item)
	}

	total := uint64(len(items))
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		items = []*financeV1.JournalEntry{}
	} else if start+pageSize < len(items) {
		items = items[start : start+pageSize]
	} else {
		items = items[start:]
	}
	return &financeV1.ListJournalEntryResponse{Total: total, Items: items}, nil
}

// TrialBalance 科目余额表：全科目（含零余额）× 借贷累计 × 方向后净额；
// 借贷合计恒等由过账平衡保证（此处再校验一次供报表呈现）。
func (r *JournalRepo) TrialBalance(
	ctx context.Context,
	req *financeV1.GetTrialBalanceRequest,
) (*financeV1.TrialBalanceResponse, error) {
	tid, _ := maybeTenantFromViewer(ctx)

	// 范围内凭证集合（日期过滤在 Go 侧——entry_date 可空，口径与流水页一致）。
	entryRows, err := r.entClient.Client().JournalEntry.Query().
		Where(journalentry.TenantIDEQ(tid)).
		All(ctx)
	if err != nil {
		return nil, financeV1.ErrorInternalServerError("query journal entries failed")
	}
	entryIDs := make([]uint32, 0, len(entryRows))
	for _, e := range entryRows {
		if e.EntryDate == nil {
			continue
		}
		if req.GetFromDate() != nil && e.EntryDate.Before(req.GetFromDate().AsTime()) {
			continue
		}
		if req.GetToDate() != nil && e.EntryDate.After(req.GetToDate().AsTime()) {
			continue
		}
		entryIDs = append(entryIDs, e.ID)
	}
	if len(entryIDs) == 0 {
		return r.emptyTrialBalance(ctx)
	}

	var aggRows []struct {
		AccountCode string `sql:"account_code"`
		Debit       int64  `sql:"debit"`
		Credit      int64  `sql:"credit"`
	}
	if err := r.entClient.Client().JournalLine.Query().
		Where(
			journalline.TenantIDEQ(tid),
			journalline.EntryIDIn(entryIDs...),
		).
		Modify(func(se *sql.Selector) {
			se.Select(
				sql.As(se.C(journalline.FieldAccountCode), "account_code"),
				sql.As(sql.Sum(se.C(journalline.FieldDebit)), "debit"),
				sql.As(sql.Sum(se.C(journalline.FieldCredit)), "credit"),
			)
			se.GroupBy(se.C(journalline.FieldAccountCode))
		}).
		Scan(ctx, &aggRows); err != nil {
		r.log.Errorf("aggregate trial balance failed: %s", err.Error())
		return nil, financeV1.ErrorInternalServerError("aggregate trial balance failed")
	}

	sumByCode := map[string]*financeV1.TrialBalanceItem{}
	for i := range aggRows {
		row := aggRows[i]
		sumByCode[row.AccountCode] = &financeV1.TrialBalanceItem{
			AccountCode: &row.AccountCode,
			DebitTotal:  &row.Debit,
			CreditTotal: &row.Credit,
		}
	}

	accounts, err := r.accountRepo.AllAccounts(ctx)
	if err != nil {
		return nil, err
	}
	resp := &financeV1.TrialBalanceResponse{Items: []*financeV1.TrialBalanceItem{}}
	var totalDebit, totalCredit int64
	for code, acc := range accounts {
		item := sumByCode[code]
		if item == nil {
			zero := int64(0)
			item = &financeV1.TrialBalanceItem{AccountCode: &code, DebitTotal: &zero, CreditTotal: &zero}
		}
		name, cat, dir := acc.GetName(), acc.GetCategory(), acc.GetBalanceDirection()
		item.AccountName, item.Category, item.BalanceDirection = &name, &cat, &dir
		// 余额 = 方向后净额：DEBIT→借−贷，CREDIT→贷−借。
		balance := item.GetDebitTotal() - item.GetCreditTotal()
		if dir == "CREDIT" {
			balance = item.GetCreditTotal() - item.GetDebitTotal()
		}
		b := balance
		item.Balance = &b

		resp.Items = append(resp.Items, item)
		totalDebit += item.GetDebitTotal()
		totalCredit += item.GetCreditTotal()
	}
	resp.TotalDebit, resp.TotalCredit = &totalDebit, &totalCredit
	return resp, nil
}

func (r *JournalRepo) emptyTrialBalance(ctx context.Context) (*financeV1.TrialBalanceResponse, error) {
	accounts, err := r.accountRepo.AllAccounts(ctx)
	if err != nil {
		return nil, err
	}
	resp := &financeV1.TrialBalanceResponse{Items: []*financeV1.TrialBalanceItem{}}
	zero := int64(0)
	for code, acc := range accounts {
		name, cat, dir := acc.GetName(), acc.GetCategory(), acc.GetBalanceDirection()
		resp.Items = append(resp.Items, &financeV1.TrialBalanceItem{
			AccountCode:       &code,
			AccountName:       &name,
			Category:          &cat,
			BalanceDirection:  &dir,
			DebitTotal:        &zero,
			CreditTotal:       &zero,
			Balance:           &zero,
		})
	}
	td, tc := int64(0), int64(0)
	resp.TotalDebit, resp.TotalCredit = &td, &tc
	return resp, nil
}


// AccountNetInRange 区间内单科目借/贷累计（驾驶舱汇总用；日期过滤与
// 余额表同口径）。
func (r *JournalRepo) AccountNetInRange(
	ctx context.Context,
	accountCode string,
	from, to time.Time,
) (debit, credit int64, err error) {
	tid, _ := maybeTenantFromViewer(ctx)

	entryRows, err := r.entClient.Client().JournalEntry.Query().
		Where(journalentry.TenantIDEQ(tid)).
		All(ctx)
	if err != nil {
		return 0, 0, financeV1.ErrorInternalServerError("query journal entries failed")
	}
	ids := make([]uint32, 0, len(entryRows))
	for _, e := range entryRows {
		if e.EntryDate == nil {
			continue
		}
		if e.EntryDate.Before(from) || e.EntryDate.After(to) {
			continue
		}
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return 0, 0, nil
	}

	var agg []struct {
		Debit  int64 `sql:"debit"`
		Credit int64 `sql:"credit"`
	}
	if err := r.entClient.Client().JournalLine.Query().
		Where(
			journalline.TenantIDEQ(tid),
			journalline.AccountCodeEQ(accountCode),
			journalline.EntryIDIn(ids...),
		).
		Modify(func(se *sql.Selector) {
			se.Select(
				sql.As(sql.Sum(se.C(journalline.FieldDebit)), "debit"),
				sql.As(sql.Sum(se.C(journalline.FieldCredit)), "credit"),
			)
		}).
		Scan(ctx, &agg); err != nil {
		r.log.Errorf("aggregate account net failed: %s", err.Error())
		return 0, 0, financeV1.ErrorInternalServerError("aggregate account net failed")
	}
	if len(agg) == 0 {
		return 0, 0, nil
	}
	return agg[0].Debit, agg[0].Credit, nil
}
