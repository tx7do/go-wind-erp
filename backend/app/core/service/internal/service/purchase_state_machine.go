package service

import (
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// purchaseOrderStatusTransition defines the allowed transitions for the
// purchase order state machine. DRAFT/SUBMITTED/APPROVED are live states;
// REJECTED/COMPLETED/CANCELLED are terminal for flow purposes (REJECTED may
// only go to CANCELLED for cleanup). Same-state transitions are rejected
// (re-submitting a submitted order is a no-op that must fail loudly).
var purchaseOrderStatusTransition = map[procurementV1.PurchaseOrder_Status][]procurementV1.PurchaseOrder_Status{
	procurementV1.PurchaseOrder_DRAFT: {
		procurementV1.PurchaseOrder_SUBMITTED,
		procurementV1.PurchaseOrder_CANCELLED,
	},
	procurementV1.PurchaseOrder_SUBMITTED: {
		procurementV1.PurchaseOrder_APPROVED,
		procurementV1.PurchaseOrder_REJECTED,
		procurementV1.PurchaseOrder_CANCELLED,
	},
	procurementV1.PurchaseOrder_APPROVED: {
		procurementV1.PurchaseOrder_COMPLETED,
		procurementV1.PurchaseOrder_CANCELLED,
	},
	procurementV1.PurchaseOrder_REJECTED: {
		procurementV1.PurchaseOrder_CANCELLED,
	},
}

// validatePurchaseOrderTransition returns true when transitioning from
// `from` to `to` is permitted. Transitions out of terminal states
// (COMPLETED/CANCELLED) and same-state transitions are rejected.
func validatePurchaseOrderTransition(from, to procurementV1.PurchaseOrder_Status) bool {
	if from == to {
		return false
	}
	allowed, ok := purchaseOrderStatusTransition[from]
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
