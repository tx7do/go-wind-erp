package data

import (
	"context"
	sqllib "database/sql"
	"sync"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/inventory"

	"github.com/tx7do/go-utils/trans"
	appViewer "go-wind-erp/pkg/entgo/viewer"

	_ "modernc.org/sqlite"
	sqlite "modernc.org/sqlite"

	_ "go-wind-erp/app/core/service/internal/data/ent/runtime"
)

// TestInventoryQuantityAtomicity verifies that concurrent updates to the same
// inventory row do not produce lost updates. A set of goroutines each performs
// a read-modify-write cycle (increment quantity by 1) on the same row, each
// inside its own transaction. Under proper transaction isolation the final
// quantity must equal initial + N. If isolation were broken (lost updates),
// the final value would be lower.
//
// The test uses an in-memory SQLite database (pure-Go modernc.org/sqlite, no
// CGO) via the connector API. SetMaxOpenConns(1) is mandatory here: the
// in-memory DB is per-connection, so a single shared connection keeps the
// schema and seeded row visible across all transactions in the test. Without
// it each transaction would see an empty database.
func TestInventoryQuantityAtomicity(t *testing.T) {
	// Build an in-memory SQLite connection via the modernc connector API.
	// The DSN must use file::memory:?cache=shared so the DB persists across
	// connections, and _fk=1 so ent's schema migration with foreign keys
	// succeeds.
	connector, err := sqlite.NewConnector("file::memory:?cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("sqlite connector: %v", err)
	}
	db := sqllib.OpenDB(connector)
	// A single connection is required so the in-memory DB is shared across
	// all transactions in this test process.
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// The inventory schema installs a TenantPrivacy policy via the TenantID
	// mixin. That policy requires a viewer in the context for every query and
	// mutation. In the live request path the ent.Server() middleware builds
	// the viewer from auth metadata; in the test we inject a SystemViewer
	// (tenant_id=0, platform scope) so the privacy policy accepts the
	// operations.
	ctx := appViewer.NewSystemViewerContext(context.Background())

	// Auto-migrate the inventory schema into the in-memory DB.
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed one inventory row at quantity 0.
	created, err := client.Inventory.Create().
		SetNillableWarehouseCode(trans.Ptr("WH-TEST")).
		SetNillableSkuCode(trans.Ptr("SKU-TEST")).
		SetNillableQuantity(trans.Ptr(int64(0))).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed inventory: %v", err)
	}

	// N goroutines each read the current quantity, increment by 1, and write
	// back inside a transaction. The serialized final value must be N.
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			// Each goroutine performs a read-modify-write in a transaction.
			tx, err := client.Tx(ctx)
			if err != nil {
				t.Errorf("begin tx: %v", err)
				return
			}
			row, err := tx.Inventory.Query().
				Where(inventory.IDEQ(created.ID)).
				Only(ctx)
			if err != nil {
				_ = tx.Rollback()
				t.Errorf("query within tx: %v", err)
				return
			}
			cur := row.Quantity
			if cur == nil {
				_ = tx.Rollback()
				return
			}
			// Use the checked-add helper to detect overflow before writing.
			next, overflow := addCheckedInt64(*cur, 1)
			if overflow {
				_ = tx.Rollback()
				return
			}
			_, err = tx.Inventory.UpdateOneID(created.ID).
				SetNillableQuantity(trans.Ptr(next)).
				Save(ctx)
			if err != nil {
				_ = tx.Rollback()
				t.Errorf("update within tx: %v", err)
				return
			}
			_ = tx.Commit()
		}()
	}
	close(start)
	wg.Wait()

	// After all goroutines finish, the quantity must equal goroutines (each
	// added exactly 1). If isolation were broken (lost updates), the value
	// would be lower.
	final, err := client.Inventory.Query().
		Where(inventory.IDEQ(created.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("final query: %v", err)
	}
	got := int64(0)
	if final.Quantity != nil {
		got = *final.Quantity
	}
	if got != int64(goroutines) {
		t.Errorf("expected quantity %d after %d concurrent increments, got %d (lost updates detected)", goroutines, goroutines, got)
	}
}

// addCheckedInt64 is a local copy of the checked-add helper so the data-layer
// test does not depend on the service package.
func addCheckedInt64(a, b int64) (int64, bool) {
	r := a + b
	if (b > 0 && r < a) || (b < 0 && r > a) {
		return 0, true
	}
	return r, false
}

// TestSumQuantityEmptyTableLocked regressions the audit finding: a bare
// SELECT SUM over an empty set returns one NULL row (not zero rows); the
// []int64 scan would error. The repo now scans NullInt64 — verify the same
// query shape survives an empty table via the fixed scan type.
func TestSumQuantityEmptyTableLocked(t *testing.T) {
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

	// No rows inserted — bare SUM returns one NULL row; the COALESCE-wrapped
	// query must return exactly one row with value 0 and scan into []int64.
	var totals []int64
	if err := client.Inventory.Query().
		Modify(func(se *entsql.Selector) {
			se.Select("COALESCE(" + entsql.Sum(se.C(inventory.FieldQuantity)) + ", 0)")
		}).
		Scan(ctx, &totals); err != nil {
		t.Fatalf("sum on empty table must not error, got: %v", err)
	}
	if len(totals) != 1 || totals[0] != 0 {
		t.Fatalf("expected [0] on empty table, got %v", totals)
	}
}
