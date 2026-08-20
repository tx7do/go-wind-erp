package service

import (
	"testing"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// TestApprovalTransition_Authorized verifies the documented transitions:
// PENDING may go to APPROVED / REJECTED / CANCELLED.
func TestApprovalTransition_Authorized(t *testing.T) {
	authorized := []struct {
		from, to approvalV1.ApprovalRequest_Status
	}{
		{approvalV1.ApprovalRequest_PENDING, approvalV1.ApprovalRequest_APPROVED},
		{approvalV1.ApprovalRequest_PENDING, approvalV1.ApprovalRequest_REJECTED},
		{approvalV1.ApprovalRequest_PENDING, approvalV1.ApprovalRequest_CANCELLED},
	}
	for _, c := range authorized {
		if !validateApprovalTransition(c.from, c.to) {
			t.Errorf("expected transition %v -> %v to be allowed", c.from, c.to)
		}
	}
}

// TestApprovalTransition_Unauthorized verifies that transitions out of
// terminal states are rejected, including "re-approving" an already-approved
// request and any cross-terminal jump.
func TestApprovalTransition_Unauthorized(t *testing.T) {
	statuses := []approvalV1.ApprovalRequest_Status{
		approvalV1.ApprovalRequest_PENDING,
		approvalV1.ApprovalRequest_APPROVED,
		approvalV1.ApprovalRequest_REJECTED,
		approvalV1.ApprovalRequest_CANCELLED,
	}
	for _, from := range statuses {
		for _, to := range statuses {
			// The only legal transitions are PENDING → {APPROVED, REJECTED, CANCELLED}.
			legal := from == approvalV1.ApprovalRequest_PENDING &&
				to != approvalV1.ApprovalRequest_PENDING
			if validateApprovalTransition(from, to) != legal {
				t.Errorf("transition %v -> %v: expected allowed=%v", from, to, legal)
			}
		}
	}
}

// TestApprovalTransition_SameState verifies same-state transitions are
// rejected (unlike the inventory machine, a no-op approve must fail).
func TestApprovalTransition_SameState(t *testing.T) {
	statuses := []approvalV1.ApprovalRequest_Status{
		approvalV1.ApprovalRequest_PENDING,
		approvalV1.ApprovalRequest_APPROVED,
		approvalV1.ApprovalRequest_REJECTED,
		approvalV1.ApprovalRequest_CANCELLED,
	}
	for _, s := range statuses {
		if validateApprovalTransition(s, s) {
			t.Errorf("expected same-state transition %v -> %v to be rejected", s, s)
		}
	}
}
