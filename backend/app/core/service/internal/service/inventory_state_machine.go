package service

import (
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// inventoryStatusTransition defines the allowed transitions for the
// inventory status state machine. The status of an inventory record may
// only change along these directed edges. Any transition not present here
// is rejected by validateStatusTransition.
var inventoryStatusTransition = map[inventoryV1.Inventory_Status][]inventoryV1.Inventory_Status{
	inventoryV1.Inventory_AVAILABLE: {
		inventoryV1.Inventory_LOCKED,
		inventoryV1.Inventory_QUARANTINED,
	},
	inventoryV1.Inventory_LOCKED: {
		inventoryV1.Inventory_AVAILABLE,
	},
	inventoryV1.Inventory_QUARANTINED: {
		inventoryV1.Inventory_AVAILABLE,
	},
}

// validateStatusTransition returns true when transitioning from `from` to `to`
// is permitted by the inventory state machine. A transition from the same
// status to itself is always allowed (no-op update). The zero value (AVAILABLE,
// enum default 0) to any status is allowed on initial creation.
func validateStatusTransition(from, to inventoryV1.Inventory_Status) bool {
	if from == to {
		return true
	}
	allowed, ok := inventoryStatusTransition[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
