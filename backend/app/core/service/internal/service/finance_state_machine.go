package service

import (
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// payableStatusTransition defines the allowed transitions for the payable
// state machine. Payment drives PENDING→PARTIAL→SETTLED (via ApplyPayment's
// CASE expression); Cancel is a guarded admin action (PENDING-unpaid only).
// PARTIAL cannot be cancelled (paid money exists — settle or adjust instead).
var payableStatusTransition = map[financeV1.Payable_Status][]financeV1.Payable_Status{
	financeV1.Payable_PENDING: {
		financeV1.Payable_PARTIAL,
		financeV1.Payable_SETTLED,
		financeV1.Payable_CANCELLED,
	},
	financeV1.Payable_PARTIAL: {
		financeV1.Payable_SETTLED,
	},
}

// validatePayableTransition returns true when transitioning from `from` to
// `to` is permitted. SETTLED/CANCELLED are terminal; same-state rejected.
func validatePayableTransition(from, to financeV1.Payable_Status) bool {
	if from == to {
		return false
	}
	allowed, ok := payableStatusTransition[from]
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
