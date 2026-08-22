package service

import (
	"testing"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// TestStockMoveTransition_Authorized verifies the 4 documented edges:
// DRAFT→{CONFIRMED,CANCELLED}, CONFIRMED→{DONE,CANCELLED}.
func TestStockMoveTransition_Authorized(t *testing.T) {
	authorized := []struct {
		from, to inventoryV1.StockMove_State
	}{
		{inventoryV1.StockMove_DRAFT, inventoryV1.StockMove_CONFIRMED},
		{inventoryV1.StockMove_DRAFT, inventoryV1.StockMove_CANCELLED},
		{inventoryV1.StockMove_CONFIRMED, inventoryV1.StockMove_DONE},
		{inventoryV1.StockMove_CONFIRMED, inventoryV1.StockMove_CANCELLED},
	}
	for _, c := range authorized {
		if !validateStockMoveTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be allowed", c.from, c.to)
		}
	}
}

// TestStockMoveTransition_Unauthorized full-matrix: only the 4 legal edges
// pass; everything else (incl. terminal exits DONE/CANCELLED, same-state,
// and cross-jumps like DRAFT→DONE bypassing confirm) is rejected.
func TestStockMoveTransition_Unauthorized(t *testing.T) {
	states := []inventoryV1.StockMove_State{
		inventoryV1.StockMove_DRAFT,
		inventoryV1.StockMove_CONFIRMED,
		inventoryV1.StockMove_DONE,
		inventoryV1.StockMove_CANCELLED,
	}
	legal := map[inventoryV1.StockMove_State]map[inventoryV1.StockMove_State]bool{
		inventoryV1.StockMove_DRAFT: {
			inventoryV1.StockMove_CONFIRMED: true,
			inventoryV1.StockMove_CANCELLED: true,
		},
		inventoryV1.StockMove_CONFIRMED: {
			inventoryV1.StockMove_DONE:      true,
			inventoryV1.StockMove_CANCELLED: true,
		},
	}
	for _, from := range states {
		for _, to := range states {
			want := legal[from] != nil && legal[from][to]
			if validateStockMoveTransition(from, to) != want {
				t.Errorf("transition %v -> %v: expected allowed=%v", from, to, want)
			}
		}
	}
}
