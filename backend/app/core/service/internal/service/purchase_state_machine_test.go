package service

import (
	"testing"

	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// TestPurchaseOrderTransition_Authorized verifies documented edges:
// DRAFT→{SUBMITTED,CANCELLED}, SUBMITTED→{APPROVED,REJECTED,CANCELLED},
// APPROVED→{COMPLETED,CANCELLED}, REJECTED→CANCELLED.
func TestPurchaseOrderTransition_Authorized(t *testing.T) {
	authorized := []struct {
		from, to procurementV1.PurchaseOrder_Status
	}{
		{procurementV1.PurchaseOrder_DRAFT, procurementV1.PurchaseOrder_SUBMITTED},
		{procurementV1.PurchaseOrder_DRAFT, procurementV1.PurchaseOrder_CANCELLED},
		{procurementV1.PurchaseOrder_SUBMITTED, procurementV1.PurchaseOrder_APPROVED},
		{procurementV1.PurchaseOrder_SUBMITTED, procurementV1.PurchaseOrder_REJECTED},
		{procurementV1.PurchaseOrder_SUBMITTED, procurementV1.PurchaseOrder_CANCELLED},
		{procurementV1.PurchaseOrder_APPROVED, procurementV1.PurchaseOrder_COMPLETED},
		{procurementV1.PurchaseOrder_APPROVED, procurementV1.PurchaseOrder_CANCELLED},
		{procurementV1.PurchaseOrder_REJECTED, procurementV1.PurchaseOrder_CANCELLED},
		{procurementV1.PurchaseOrder_REJECTED, procurementV1.PurchaseOrder_SUBMITTED},
	}
	for _, c := range authorized {
		if !validatePurchaseOrderTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be allowed", c.from, c.to)
		}
	}
}

// TestPurchaseOrderTransition_Unauthorized full-matrix: only the 9 legal
// edges pass; everything else (incl. terminal exits and cross-jumps like
// DRAFT→APPROVED bypassing submission) is rejected.
func TestPurchaseOrderTransition_Unauthorized(t *testing.T) {
	statuses := []procurementV1.PurchaseOrder_Status{
		procurementV1.PurchaseOrder_DRAFT,
		procurementV1.PurchaseOrder_SUBMITTED,
		procurementV1.PurchaseOrder_APPROVED,
		procurementV1.PurchaseOrder_REJECTED,
		procurementV1.PurchaseOrder_COMPLETED,
		procurementV1.PurchaseOrder_CANCELLED,
	}
	legal := map[procurementV1.PurchaseOrder_Status]map[procurementV1.PurchaseOrder_Status]bool{
		procurementV1.PurchaseOrder_DRAFT: {
			procurementV1.PurchaseOrder_SUBMITTED: true, procurementV1.PurchaseOrder_CANCELLED: true,
		},
		procurementV1.PurchaseOrder_SUBMITTED: {
			procurementV1.PurchaseOrder_APPROVED: true, procurementV1.PurchaseOrder_REJECTED: true,
			procurementV1.PurchaseOrder_CANCELLED: true,
		},
		procurementV1.PurchaseOrder_APPROVED: {
			procurementV1.PurchaseOrder_COMPLETED: true, procurementV1.PurchaseOrder_CANCELLED: true,
		},
		procurementV1.PurchaseOrder_REJECTED: {
			procurementV1.PurchaseOrder_CANCELLED: true,
			procurementV1.PurchaseOrder_SUBMITTED: true,
		},
	}
	for _, from := range statuses {
		for _, to := range statuses {
			want := legal[from] != nil && legal[from][to]
			if validatePurchaseOrderTransition(from, to) != want {
				t.Errorf("transition %v -> %v: expected allowed=%v", from, to, want)
			}
		}
	}
}

// TestMulChecked mirrors money overflow guards for the procurement amounts.
func TestMulChecked(t *testing.T) {
	if got, of := mulChecked(3, 5); of || got != 15 {
		t.Errorf("mulChecked(3,5) = %d, %v", got, of)
	}
	if got, of := mulChecked(0, 5); of || got != 0 {
		t.Errorf("mulChecked(0,5) = %d, %v", got, of)
	}
	// MaxInt64 × 2 must overflow.
	if _, of := mulChecked(9223372036854775807, 2); !of {
		t.Errorf("mulChecked(MaxInt64,2): expected overflow")
	}
	// A realistic large order: 1e6 units × 1e6 cents (1万元/件 × 百万件) fits.
	if got, of := mulChecked(1_000_000, 1_000_000); of || got != 1_000_000_000_000 {
		t.Errorf("mulChecked(1e6,1e6) = %d, %v", got, of)
	}
}
