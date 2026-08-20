package service

import (
	"testing"

	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
)

// TestPayableTransition_Authorized verifies the documented edges:
// PENDING→{PARTIAL,SETTLED,CANCELLED}, PARTIAL→SETTLED.
func TestPayableTransition_Authorized(t *testing.T) {
	authorized := []struct {
		from, to financeV1.Payable_Status
	}{
		{financeV1.Payable_PENDING, financeV1.Payable_PARTIAL},
		{financeV1.Payable_PENDING, financeV1.Payable_SETTLED},
		{financeV1.Payable_PENDING, financeV1.Payable_CANCELLED},
		{financeV1.Payable_PARTIAL, financeV1.Payable_SETTLED},
	}
	for _, c := range authorized {
		if !validatePayableTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be allowed", c.from, c.to)
		}
	}
}

// TestPayableTransition_Unauthorized full-matrix: only the 4 legal edges
// pass — terminal exits, PARTIAL→CANCELLED (paid money cannot vanish), and
// same-state transitions are all rejected.
func TestPayableTransition_Unauthorized(t *testing.T) {
	statuses := []financeV1.Payable_Status{
		financeV1.Payable_PENDING,
		financeV1.Payable_PARTIAL,
		financeV1.Payable_SETTLED,
		financeV1.Payable_CANCELLED,
	}
	legal := map[financeV1.Payable_Status]map[financeV1.Payable_Status]bool{
		financeV1.Payable_PENDING: {
			financeV1.Payable_PARTIAL: true, financeV1.Payable_SETTLED: true,
			financeV1.Payable_CANCELLED: true,
		},
		financeV1.Payable_PARTIAL: {
			financeV1.Payable_SETTLED: true,
		},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			want := legal[from] != nil && legal[from][to]
			if validatePayableTransition(from, to) != want {
				t.Errorf("transition %v -> %v: expected allowed=%v", from, to, want)
			}
		}
	}
}
