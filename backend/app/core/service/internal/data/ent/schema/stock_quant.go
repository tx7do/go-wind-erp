package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
	appmixin "go-wind-erp/pkg/entgo/mixin"
)

// StockQuant holds the schema definition for the StockQuant entity.
//
// 库存量（借鉴 Odoo stock.quant：按 location+product 自然键存储在手量）。
// quantity 是库存唯一真相，仅由 StockPickingService.Validate 通过创建
// StockMoveLine 来变更。无 status 字段（Odoo 无此概念，隔离用 location 建模）。
type StockQuant struct {
	ent.Schema
}

func (StockQuant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_quants",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存量表"),
	}
}

func (StockQuant) Fields() []ent.Field {
	return []ent.Field{
		// location_id：该量所在的位置（仅 INTERNAL）。
		field.Uint32("location_id").
			Comment("所在位置ID").
			Default(0).
			Optional().
			Nillable(),

		field.String("product_code").
			Comment("产品编码").
			Optional().
			Nillable(),

		field.Int64("quantity").
			Comment("在手数量").
			Default(0).
			Optional().
			Nillable(),

		// cost_price：加权平均成本（分）。入库时按采购单价做加权平均更新，
		// 出库时冻结到 stock_move_line.unit_cost 用于 COGS 核算。
		field.Int64("cost_price").
			Comment("加权平均成本（分）").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (StockQuant) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockQuant) Indexes() []ent.Index {
	return []ent.Index{
		// 按 location+product 自然键唯一（Odoo stock.quant 的自然键约束）
		index.Fields("tenant_id", "location_id", "product_code").
			Unique().
			StorageKey("idx_inv_stock_quant_tenant_location_product"),
	}
}
