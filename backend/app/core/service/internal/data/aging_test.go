package data

import (
	"context"
	sqllib "database/sql"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/payable"
	appViewer "go-wind-erp/pkg/entgo/viewer"

	_ "modernc.org/sqlite"
	sqlite "modernc.org/sqlite"

	_ "go-wind-erp/app/core/service/internal/data/ent/runtime"
)

// TestAgingReportOutstandingAndCount asserts two correctness properties of
// PayableRepo.AgingReport that a prior implementation broke:
//
//  1. The per-bucket total must be the sum of OUTSTANDING balances
//     (amount − paid_amount), NOT the sum of face-value amounts. The old code
//     summed `amount`, so a partially-paid (PARTIAL) row inflated the bucket
//     by its already-paid portion.
//  2. The per-bucket count must ACCUMULATE the number of rows landing in the
//     bucket. The old code overwrote Count to 1 each iteration, so a bucket
//     with N rows reported 1 (or nil) regardless of N.
//
// Seed: two PARTIAL rows in the same time bucket (deterministic overdue),
// face amounts 1000 and 2000 with paid 300 and 500 → outstanding 700 and 1500.
// Expected for that bucket: count 2, total 2200 (NOT face-value 3000).
func TestAgingReportOutstandingAndCount(t *testing.T) {
	// --- in-memory SQLite setup (mirrors stock_test.go) ---
	connector, err := sqlite.NewConnector("file::memory:?cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("sqlite connector: %v", err)
	}
	db := sqllib.OpenDB(connector)
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	ctx := appViewer.NewSystemViewerContext(context.Background())

	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// --- seed two PARTIAL payables in the overdue bucket ---
	overdueDate := time.Now().AddDate(0, 0, -30) // firmly overdue
	rows := []struct{ amount, paid int64 }{
		{amount: 1000, paid: 300},
		{amount: 2000, paid: 500},
	}
	for _, r := range rows {
		status := payable.StatusPartial
		if err := client.Payable.Create().
			SetTenantID(0).
			SetNillableSupplierCode(trans.Ptr("SUP-TEST")).
			SetAmount(r.amount).
			SetPaidAmount(r.paid).
			SetDueDate(overdueDate).
			SetStatus(status).
			Exec(ctx); err != nil {
			t.Fatalf("seed payable: %v", err)
		}
	}

	// --- build a minimal PayableRepo wired only to the ent client + logger ---
	// AgingReport uses solely r.entClient and r.log, so the mapper/repository
	// fields can stay nil. EntClient wraps the raw client via NewEntClient.
	entClient := entCrud.NewEntClient[*ent.Client](client, drv)
	repo := &PayableRepo{
		entClient: entClient,
		log:       log.NewHelper(log.NewStdLogger(os.Stderr)),
	}

	buckets, err := repo.AgingReport(ctx)
	if err != nil {
		t.Fatalf("aging report: %v", err)
	}

	// Locate the overdue bucket by label.
	var overdueCount, overdueTotal int64
	found := false
	for _, b := range buckets {
		if b.GetBucket() == "overdue" {
			found = true
			if b.Count != nil {
				overdueCount = *b.Count
			}
			if b.TotalAmount != nil {
				overdueTotal = *b.TotalAmount
			}
		}
	}
	if !found {
		t.Fatalf("overdue bucket missing from report")
	}

	// Property 1: total must be outstanding balances (700 + 1500 = 2200),
	// NOT face value (1000 + 2000 = 3000).
	const wantOutstanding int64 = 2200
	if overdueTotal != wantOutstanding {
		t.Errorf("overdue bucket total = %d; want %d (outstanding sum, not face-value %d)",
			overdueTotal, wantOutstanding, int64(3000))
	}

	// Property 2: count must equal the number of rows (2), not be stuck at 1.
	const wantCount int64 = 2
	if overdueCount != wantCount {
		t.Errorf("overdue bucket count = %d; want %d (accumulated, not overwrite-1)", overdueCount, wantCount)
	}
}

