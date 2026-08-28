package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// StockPicking holds the schema definition for the StockPicking entity.
//
// 拣货单（借鉴 Odoo stock.picking：一等文档，有自己的生命周期）。
// 不存储 state——状态从子 StockMove 派生（服务层每次读取时聚合）。
// source/dest location 由服务层按 picking_type + 仓库推导，客户端不提供。
// purchase_order_id 仅入库拣货单有值（Odoo PO→receipt 桥接）。
type StockPicking struct {
	ent.Schema
}

func (StockPicking) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_pickings",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("拣货单表"),
	}
}

func (StockPicking) Fields() []ent.Field {
	return []ent.Field{
		field.String("picking_number").
			Comment("拣货单号（服务端生成）").
			Optional().
			Nillable(),

		// picking_type：操作类别（Odoo picking_type.code，简化为枚举）。
		// INCOMING=入库（SUPPLIER→INTERNAL），INTERNAL=调拨（INTERNAL→INTERNAL）。
		field.Enum("picking_type").
			Comment("拣货类型：INCOMING=入库，INTERNAL=调拨，OUTGOING=出库").
			NamedValues(
				"Incoming", "INCOMING",
				"Internal", "INTERNAL",
				"Outgoing", "OUTGOING",
			).
			Default("INCOMING").
			Optional().
			Nillable(),

		// source/dest location：由服务层按 type 推导落库。
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

		// purchase_order_id：入库拣货单关联的采购单（Odoo PO→receipt 桥接）。
		field.Uint32("purchase_order_id").
			Comment("关联采购单ID（仅入库）").
			Default(0).
			Optional().
			Nillable(),

		field.String("partner_code").
			Comment("交易方编码（供应商/客户）").
			Optional().
			Nillable(),
	}
}

func (StockPicking) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockPicking) Indexes() []ent.Index {
	return []ent.Index{
		// 按创建时间分页
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_inv_stock_picking_tenant_created_at"),

		// 按采购单回溯收货拣货单
		index.Fields("tenant_id", "purchase_order_id").
			StorageKey("idx_inv_stock_picking_tenant_po"),
	}
}
