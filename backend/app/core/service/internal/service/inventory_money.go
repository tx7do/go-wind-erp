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

// mulChecked performs a checked int64 multiplication, returning the result
// and whether the operation overflowed. Purchase line amounts are
// quantity × unit-price in cents; an overflow here would corrupt the order
// total, so it must be detected and rejected rather than silently wrapped.
func mulChecked(a, b int64) (result int64, overflow bool) {
	if a == 0 || b == 0 {
		return 0, false
	}
	r := a * b
	if r/b != a {
		return 0, true
	}
	return r, false
}
