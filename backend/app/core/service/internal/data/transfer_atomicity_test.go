package data

import (
	"context"
	sqllib "database/sql"
	"errors"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/inventory"
	appViewer "go-wind-erp/pkg/entgo/viewer"

	_ "modernc.org/sqlite"
	sqlite "modernc.org/sqlite"

	_ "go-wind-erp/app/core/service/internal/data/ent/runtime"
)

// TestTransferAtomicity_RollbackOnFailure verifies the transaction contract
// that StockMovementService.Transfer now relies on: when the second leg of a
// transfer fails, the first leg's writes are rolled back by FinishTx, leaving
// source inventory unchanged.
//
// The previous Transfer used SAGA-style compensation (commit outbound, then
// try inbound; if inbound failed, try a reverse movement; if that reverse
// also failed, only log). That left a realizable window where source was
// decremented but destination never incremented. The fix wraps both legs in a
// single DB transaction so either both commit or both roll back.
//
// This test exercises the mechanism directly: begin tx, apply a delta to the
// source inside the tx, then invoke FinishTx with a non-nil error and confirm
// the source's quantity outside the tx is unchanged. This is the exact
// rollback path Transfer hits when its inbound leg fails.
func TestTransferAtomicity_RollbackOnFailure(t *testing.T) {
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

	// Seed a source inventory row at quantity 100.
	created, err := client.Inventory.Create().
		SetNillableWarehouseCode(trans.Ptr("WH-SRC")).
		SetNillableSkuCode(trans.Ptr("SKU-X")).
		SetNillableQuantity(trans.Ptr(int64(100))).
		SetStatus(inventory.StatusAvailable).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed inventory: %v", err)
	}

	// Build a minimal InventoryRepo wired only to the ent client + logger.
	// init() sets up the mapper (no external deps). BeginTx/FinishTx/ApplyDeltaTx
	// use solely r.entClient and r.log.
	entClient := entCrud.NewEntClient[*ent.Client](client, drv)
	repo := &InventoryRepo{
		entClient: entClient,
		log:       log.NewHelper(log.NewStdLogger(os.Stderr)),
	}
	repo.init()

	// Begin a transaction. Inside it, apply a -50 delta to the source (as a
	// transfer's outbound leg would). We then simulate the inbound leg failing
	// by passing a non-nil error to FinishTx, which must roll the tx back.
	tx, terr := repo.BeginTx(ctx)
	if terr != nil {
		t.Fatalf("begin tx: %v", terr)
	}

	if _, e := repo.ApplyDeltaTx(ctx, tx, created.ID, -50); e != nil {
		// Ensure tx is rolled back before bailing.
		repo.FinishTx(tx, e)
		t.Fatalf("apply delta in tx: %v", e)
	}

	// Simulate the second leg failing. FinishTx with a non-nil error invokes
	// tx.Rollback(), discarding the -50 delta applied above.
	simulatedFailure := errors.New("simulated inbound-leg failure")
	repo.FinishTx(tx, simulatedFailure)

	// After rollback, the source quantity must be unchanged (100). If the
	// transaction had committed instead, it would be 50.
	after, err := client.Inventory.Query().
		Where(inventory.IDEQ(created.ID)).
		Only(ctx)
	if err != nil {
		t.Fatalf("post-rollback query: %v", err)
	}
	got := int64(0)
	if after.Quantity != nil {
		got = *after.Quantity
	}
	const want int64 = 100
	if got != want {
		t.Errorf("source quantity after rollback = %d; want %d (rollback failed, delta leaked)", got, want)
	}
}

