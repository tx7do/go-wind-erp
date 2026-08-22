package service

import (
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// stockMoveStateTransition defines the allowed transitions for the stock move
// state machine (借鉴 Odoo stock.move.state 的简化版：DRAFT→CONFIRMED→DONE，
// 任何非终态→CANCELLED；DONE/CANCELLED 终态）。
//
// 这是计划移动的状态机。DRAFT = 刚创建（picking 创建时 moves 落 DRAFT）；
// CONFIRMED = 由 StockPickingService.Confirm 迁入（借鉴 Odoo action_confirm）；
// DONE = 由 StockPickingService.Validate 在事务内迁入（创建 move-line、更新
// stock_quant、回写采购收货量，借鉴 Odoo button_validate / _action_done）；
// CANCELLED = 由 StockPickingService.Cancel 迁入（借鉴 Odoo action_cancel）。
//
// 无 assigned/partially_available/waiting（预留层推迟——三模块范围内无出库销售）。
var stockMoveStateTransition = map[inventoryV1.StockMove_State][]inventoryV1.StockMove_State{
	inventoryV1.StockMove_DRAFT: {
		inventoryV1.StockMove_CONFIRMED,
		inventoryV1.StockMove_CANCELLED,
	},
	inventoryV1.StockMove_CONFIRMED: {
		inventoryV1.StockMove_DONE,
		inventoryV1.StockMove_CANCELLED,
	},
}

// validateStockMoveTransition returns true when transitioning from `from` to
// `to` is permitted. Transitions out of terminal states (DONE/CANCELLED) and
// same-state transitions are rejected.
func validateStockMoveTransition(from, to inventoryV1.StockMove_State) bool {
	if from == to {
		return false
	}
	allowed, ok := stockMoveStateTransition[from]
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
