package service

import "testing"

// TestSuggestReplenishQty 锁死补货量建议规则：补到阈值 2 倍，
// 且不低于一个阈值批次。
func TestSuggestReplenishQty(t *testing.T) {
	cases := []struct {
		current, threshold, want int64
	}{
		{9, 10, 11},   // 20-9
		{0, 10, 20},   // 20-0
		{15, 10, 10},  // 20-15=5 < 阈值 → 提到 10
		{20, 10, 10},  // 已达 2 倍阈值仍给最小批次（建议性数量）
		{0, 1, 2},     // 阈值 1
		{0, 100, 200}, // 大阈值
	}
	for _, c := range cases {
		if got := suggestReplenishQty(c.current, c.threshold); got != c.want {
			t.Errorf("suggestReplenishQty(%d, %d) = %d, want %d", c.current, c.threshold, got, c.want)
		}
	}
}
