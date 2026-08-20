package service

import (
	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// approvalStatusTransition defines the allowed transitions for the approval
// state machine. PENDING is the only non-terminal state; APPROVED / REJECTED /
// CANCELLED are terminal. Unlike the inventory status machine, a same-state
// "transition" is NOT allowed (re-approving an approved request is a no-op
// that must be rejected, not silently accepted).
var approvalStatusTransition = map[approvalV1.ApprovalRequest_Status][]approvalV1.ApprovalRequest_Status{
	approvalV1.ApprovalRequest_PENDING: {
		approvalV1.ApprovalRequest_APPROVED,
		approvalV1.ApprovalRequest_REJECTED,
		approvalV1.ApprovalRequest_CANCELLED,
	},
}

// validateApprovalTransition returns true when transitioning from `from` to
// `to` is permitted. Transitions from terminal states and same-state
// transitions are rejected.
func validateApprovalTransition(from, to approvalV1.ApprovalRequest_Status) bool {
	if from == to {
		return false
	}
	allowed, ok := approvalStatusTransition[from]
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
