package service

import (
	"math"
	"testing"
)

// TestAddChecked_Normal verifies that addChecked returns the correct sum and
// no overflow for in-range operands.
func TestAddChecked_Normal(t *testing.T) {
	cases := []struct {
		a, b, want int64
	}{
		{0, 0, 0},
		{10, 20, 30},
		{-10, 20, 10},
		{-10, -20, -30},
		{100, -50, 50},
		{math.MaxInt32, math.MaxInt32, int64(math.MaxInt32) * 2},
	}
	for _, c := range cases {
		got, overflow := addChecked(c.a, c.b)
		if overflow {
			t.Errorf("addChecked(%d, %d): unexpected overflow", c.a, c.b)
		}
		if got != c.want {
			t.Errorf("addChecked(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestAddChecked_Overflow verifies that addChecked detects int64 overflow on
// both the positive and negative extremes.
func TestAddChecked_Overflow(t *testing.T) {
	cases := []struct {
		a, b int64
	}{
		{math.MaxInt64, 1},
		{math.MaxInt64, math.MaxInt64},
		{1, math.MaxInt64},
		{math.MinInt64, -1},
		{math.MinInt64, math.MinInt64},
		{-1, math.MinInt64},
	}
	for _, c := range cases {
		got, overflow := addChecked(c.a, c.b)
		if !overflow {
			t.Errorf("addChecked(%d, %d): expected overflow, got %d", c.a, c.b, got)
		}
	}
}

// TestAddOverflow mirrors the above for the predicate form.
func TestAddOverflow(t *testing.T) {
	if !addOverflow(math.MaxInt64, 1) {
		t.Errorf("addOverflow(MaxInt64, 1): expected true")
	}
	if !addOverflow(math.MinInt64, -1) {
		t.Errorf("addOverflow(MinInt64, -1): expected true")
	}
	if addOverflow(10, 20) {
		t.Errorf("addOverflow(10, 20): expected false")
	}
}
