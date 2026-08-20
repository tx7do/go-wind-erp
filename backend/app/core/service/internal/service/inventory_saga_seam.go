package service

import (
	"context"
	"fmt"

	"go-wind-erp/app/core/service/internal/data"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// defaultLowStockThreshold 低库存阈值（分域全局缺省；后续改由字典表
// inv:low_stock_threshold 按租户覆盖）。GetOverview 的缺省与之同源。
const defaultLowStockThreshold int64 = 10

// replenishment 审批联动约定：biz_type="replenishment"，
// biz_ref="replenishment:{warehouseCode}:{skuCode}"。
const (
	replenishmentBizType = "replenishment"
	replenishmentBizRef  = "replenishment:%s:%s"
)

// stockEvent 出库事件（SAGA 反应的输入）。
type stockEvent struct {
	warehouseCode string
	skuCode       string
	delta         int64
}

// sagaSeam 库存→采购的反应缝（"SAGA"在本单体架构下的真实形态）：
// 出库后低库存 → 幂等评估 → 补货建议审批单 → 人审后生成草稿采购单。
//
// 语义约定：
//   - 评估失败不阻塞出库（调用方仅记录错误）——下次任何出库会重新
//     评估，最终一致靠"重评估"而非"重放"；
//   - 幂等靠双重在途检查（未收满的在途 PO + 未决补货审批），极端
//     并发下仍可能偶发重复建议，由审批人去重；
//   - 采购决策保留人机环：不自动下采购单。
type sagaSeam interface {
	notifyProcurement(ctx context.Context, event stockEvent) error
}

// procurementSagaSeam sagaSeam 的真实现（依赖在 NewStockMovementService
// 组装，同进程直连 repo）。
type procurementSagaSeam struct {
	inventoryRepo        *data.InventoryRepo
	purchaseOrderRepo    *data.PurchaseOrderRepo
	approvalRequestRepo  *data.ApprovalRequestRepo
}

// notifyProcurement 出库后的补货评估。
func (s *procurementSagaSeam) notifyProcurement(ctx context.Context, event stockEvent) error {
	if event.delta >= 0 {
		return nil
	}

	inv, err := s.inventoryRepo.FindByWarehouseSku(ctx, event.warehouseCode, event.skuCode)
	if err != nil {
		return fmt.Errorf("saga: query inventory: %w", err)
	}
	current := inv.GetQuantity()
	if current >= defaultLowStockThreshold {
		return nil
	}

	// 幂等检查一：该 SKU 已有在途补货（SUBMITTED/APPROVED 且未收满）。
	inFlight, err := s.purchaseOrderRepo.HasInFlightReplenishment(ctx, event.skuCode)
	if err != nil {
		return fmt.Errorf("saga: check in-flight po: %w", err)
	}
	if inFlight {
		return nil
	}

	bizRef := fmt.Sprintf(replenishmentBizRef, event.warehouseCode, event.skuCode)

	// 幂等检查二：已有未决的补货建议审批单。
	pending, err := s.approvalRequestRepo.HasPendingByBizRef(ctx, bizRef)
	if err != nil {
		return fmt.Errorf("saga: check pending proposal: %w", err)
	}
	if pending {
		return nil
	}

	// 生成补货建议审批单；供应商取该 SKU 最近采购来源（可能为空——
	// 审批通过时无供应商史则不自动建单，由采购员手工处理）。
	supplier, _ := s.purchaseOrderRepo.LastSupplierForSku(ctx, event.skuCode)
	qty := suggestReplenishQty(current, defaultLowStockThreshold)

	_, err = s.approvalRequestRepo.Create(ctx, &approvalV1.CreateApprovalRequestRequest{
		Data: &approvalV1.ApprovalRequest{
			Title:   strPtr(fmt.Sprintf("低库存补货建议 %s/%s（当前 %d）", event.warehouseCode, event.skuCode, current)),
			BizType: strPtr(replenishmentBizType),
			BizRef:  strPtr(bizRef),
			Summary: strPtr(fmt.Sprintf("建议补货 %d；供应商 %s", qty, orUnknown(supplier))),
		},
	})
	if err != nil {
		return fmt.Errorf("saga: create replenishment proposal: %w", err)
	}
	return nil
}

// suggestReplenishQty 补货量建议：补到阈值的 2 倍，且不低于一个阈值批次。
func suggestReplenishQty(current, threshold int64) int64 {
	qty := threshold*2 - current
	if qty < threshold {
		qty = threshold
	}
	return qty
}

func strPtr(s string) *string { return &s }

func orUnknown(s string) string {
	if s == "" {
		return "未知（无采购历史）"
	}
	return s
}
