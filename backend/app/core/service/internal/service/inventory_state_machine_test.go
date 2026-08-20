package service

import (
	"testing"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// TestStatusTransition_Authorized verifies that the inventory status state
// machine permits the documented transitions and rejects transitions not
// present in the transition table.
func TestStatusTransition_Authorized(t *testing.T) {
	// Authorized transitions per inventoryStatusTransition.
	authorized := []struct {
		from, to inventoryV1.Inventory_Status
	}{
		{inventoryV1.Inventory_AVAILABLE, inventoryV1.Inventory_LOCKED},
		{inventoryV1.Inventory_AVAILABLE, inventoryV1.Inventory_QUARANTINED},
		{inventoryV1.Inventory_LOCKED, inventoryV1.Inventory_AVAILABLE},
		{inventoryV1.Inventory_QUARANTINED, inventoryV1.Inventory_AVAILABLE},
	}
	for _, c := range authorized {
		if !validateStatusTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be allowed", c.from, c.to)
		}
	}
}

// TestStatusTransition_Unauthorized verifies that transitions absent from
// the transition table are rejected. These are the "unauthorized transitions"
// the state machine must block — e.g. a quarantined record jumping straight
// to locked, or a locked record being quarantined.
func TestStatusTransition_Unauthorized(t *testing.T) {
	// Unauthorized transitions: not present in inventoryStatusTransition.
	unauthorized := []struct {
		from, to inventoryV1.Inventory_Status
	}{
		{inventoryV1.Inventory_LOCKED, inventoryV1.Inventory_QUARANTINED},
		{inventoryV1.Inventory_QUARANTINED, inventoryV1.Inventory_LOCKED},
	}
	for _, c := range unauthorized {
		if validateStatusTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be rejected", c.from, c.to)
		}
	}
}

// TestStatusTransition_SameState verifies that a no-op transition (same from
// and to) is always allowed.
func TestStatusTransition_SameState(t *testing.T) {
	statuses := []inventoryV1.Inventory_Status{
		inventoryV1.Inventory_AVAILABLE,
		inventoryV1.Inventory_LOCKED,
		inventoryV1.Inventory_QUARANTINED,
	}
	for _, s := range statuses {
		if !validateStatusTransition(s, s) {
			t.Errorf("expected same-state transition %v -> %v to be allowed", s, s)
		}
	}
}
