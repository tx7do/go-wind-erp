package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// StockMoveLine holds the schema definition for the StockMoveLine entity.
//
// 执行记录（借鉴 Odoo stock.move.line 的执行角色）。记录一次实际发生的
// 库存移动：把 executed_quantity 个产品从 source 移到 dest。这是唯一能
// 变更 StockQuant.quantity 的东西（在 StockPickingService.Validate 的事务内）。
// append-only：仅 CreatedAt，无 UpdateAt。
type StockMoveLine struct {
	ent.Schema
}

func (StockMoveLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_move_lines",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存移动执行记录表"),
	}
}

func (StockMoveLine) Fields() []ent.Field {
	return []ent.Field{
		// move_id：所属计划 move。
		field.Uint32("move_id").
			Comment("所属移动计划ID").
			Default(0).
			Optional().
			Nillable(),

		// picking_id：反范式到拣货单（Odoo 式，便于按拣货单查执行记录）。
		field.Uint32("picking_id").
			Comment("所属拣货单ID（反范式）").
			Default(0).
			Optional().
			Nillable(),

		field.String("product_code").
			Comment("产品编码").
			Optional().
			Nillable(),

		field.Uint32("source_location_id").
			Comment("源位置ID").
			Default(0).
			Optional().
			Nillable(),

		field.Uint32("destination_location_id").
			Comment("目的位置ID").
			Default(0).
			Optional().
			Nillable(),

		// executed_quantity：实际执行量（Odoo qty_done）。
		field.Int64("executed_quantity").
			Comment("已执行数量").
			Default(0).
			Optional().
			Nillable(),

		// unit_cost：执行时从源位置 quant 冻结的单位成本（分）。
		// 出库腿用于 COGS 核算（executed_quantity × unit_cost）；入库腿
		// 记录采购单价供审计。
		field.Int64("unit_cost").
			Comment("执行时冻结的单位成本（分，用于COGS）").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (StockMoveLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockMoveLine) Indexes() []ent.Index {
	return []ent.Index{
		// 按 move 取其执行记录
		index.Fields("tenant_id", "move_id").
			StorageKey("idx_inv_stock_move_line_tenant_move"),

		// 按拣货单取其执行记录（反范式查询）
		index.Fields("tenant_id", "picking_id").
			StorageKey("idx_inv_stock_move_line_tenant_picking"),
	}
}
