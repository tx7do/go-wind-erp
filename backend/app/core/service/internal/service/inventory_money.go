package service

import (
	"math"
)

// addChecked performs a checked int64 addition, returning the result and
// whether the operation overflowed. The inventory quantity and stock-movement
// delta fields are monetary-quantity-like values: an overflow on addition
// (e.g. when applying a positive delta to an existing quantity) must be
// detected and rejected rather than silently wrapping, because a wrapped
// quantity would corrupt stock accounting and could allow oversell or
// negative-stock conditions.
//
// int64 addition overflows iff the signs of the operands match and the
// sign of the sum differs from the operands' sign.
func addChecked(a, b int64) (result int64, overflow bool) {
	r := a + b
	if (b > 0 && r < a) || (b < 0 && r > a) {
		return 0, true
	}
	return r, false
}

// addOverflow returns true iff the addition a+b would overflow int64.
func addOverflow(a, b int64) bool {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return true
	}
	return false
}
