package service

import (
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

// salesOrderStatusTransition defines the allowed transitions for the
// sales order state machine. DRAFT/SUBMITTED/APPROVED/REJECTED are live
// states (REJECTED supports revise-and-resubmit); CANCELLED is terminal.
// COMPLETED→APPROVED is the return-reopen edge (销售退货后单据重开继续发货，
// 由 ApplyFulfillmentReturnTx 在事务内条件迁移). Same-state transitions are
// rejected (re-submitting a submitted order is a no-op that must fail loudly).
var salesOrderStatusTransition = map[salesV1.SalesOrder_Status][]salesV1.SalesOrder_Status{
	salesV1.SalesOrder_DRAFT: {
		salesV1.SalesOrder_SUBMITTED,
		salesV1.SalesOrder_CANCELLED,
	},
	salesV1.SalesOrder_SUBMITTED: {
		salesV1.SalesOrder_APPROVED,
		salesV1.SalesOrder_REJECTED,
		salesV1.SalesOrder_CANCELLED,
	},
	salesV1.SalesOrder_APPROVED: {
		salesV1.SalesOrder_COMPLETED,
		salesV1.SalesOrder_CANCELLED,
	},
	salesV1.SalesOrder_REJECTED: {
		salesV1.SalesOrder_CANCELLED,
		// 驳回后可修改重提（改单走 Update，重提走 Submit）
		salesV1.SalesOrder_SUBMITTED,
	},
	salesV1.SalesOrder_COMPLETED: {
		// 销售退货重开（退货使 fulfilled_quantity 回落至未履约齐）
		salesV1.SalesOrder_APPROVED,
	},
}

// validateSalesOrderTransition returns true when transitioning from
// `from` to `to` is permitted. Transitions out of terminal states
// (COMPLETED/CANCELLED) and same-state transitions are rejected.
func validateSalesOrderTransition(from, to salesV1.SalesOrder_Status) bool {
	if from == to {
		return false
	}
	allowed, ok := salesOrderStatusTransition[from]
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
