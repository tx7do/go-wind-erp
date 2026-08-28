package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// StockMove holds the schema definition for the StockMove entity.
//
// 计划移动（借鉴 Odoo stock.move 的计划角色）。记录“把 N 个产品从 source
// location 移到 dest location”的意图，带状态机 DRAFT→CONFIRMED→DONE/CANCELLED。
// 与执行记录 StockMoveLine 分离——后者是唯一实际变更 StockQuant 的东西。
// purchase_order_item_id 仅入库 move 有值（Odoo purchase_line_id）。
type StockMove struct {
	ent.Schema
}

func (StockMove) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_moves",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存移动计划表"),
	}
}

func (StockMove) Fields() []ent.Field {
	return []ent.Field{
		// picking_id：父拣货单。
		field.Uint32("picking_id").
			Comment("所属拣货单ID").
			Default(0).
			Optional().
			Nillable(),

		field.String("product_code").
			Comment("产品编码").
			Optional().
			Nillable(),

		// source/dest location：移动方向（Odoo location_id/location_dest_id）。
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

		// planned_quantity：计划需求量（Odoo product_uom_qty）。
		field.Int64("planned_quantity").
			Comment("计划数量").
			Default(0).
			Optional().
			Nillable(),

		// state：状态机字段（DRAFT/CONFIRMED/DONE/CANCELLED）。
		field.Enum("state").
			Comment("移动状态").
			NamedValues(
				"Draft", "DRAFT",
				"Confirmed", "CONFIRMED",
				"Done", "DONE",
				"Cancelled", "CANCELLED",
			).
			Default("DRAFT").
			Optional().
			Nillable(),

		// purchase_order_item_id：入库 move 关联的采购明细（Odoo purchase_line_id）。
		field.Uint32("purchase_order_item_id").
			Comment("关联采购明细ID（仅入库，Odoo purchase_line_id）").
			Default(0).
			Optional().
			Nillable(),

		// sales_order_item_id：出库 move 关联的销售明细（镜像 purchase_order_item_id）。
		field.Uint32("sales_order_item_id").
			Comment("关联销售明细ID（仅出库）").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (StockMove) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockMove) Indexes() []ent.Index {
	return []ent.Index{
		// 按拣货单取其 moves
		index.Fields("tenant_id", "picking_id").
			StorageKey("idx_inv_stock_move_tenant_picking"),

		// 按状态筛选（看板/统计）
		index.Fields("tenant_id", "state").
			StorageKey("idx_inv_stock_move_tenant_state"),

		// 按采购明细回溯（PO→收货链）
		index.Fields("tenant_id", "purchase_order_item_id").
			StorageKey("idx_inv_stock_move_tenant_po_item"),

		// 按销售明细回溯（SO→出库链）
		index.Fields("tenant_id", "sales_order_item_id").
			StorageKey("idx_inv_stock_move_tenant_so_item"),
	}
}
