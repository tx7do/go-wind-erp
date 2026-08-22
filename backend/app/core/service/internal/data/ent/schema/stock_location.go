package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// StockLocation holds the schema definition for the StockLocation entity.
//
// 库存位置（借鉴 Odoo stock.location 的 usage 概念，简化为扁平结构）。
// usage 枚举驱动移动语义：SUPPLIER→INTERNAL = 入库，INTERNAL→INTERNAL = 调拨。
// parent_id/path 字段保留供未来层级扩展，初始实现为扁平（每租户一条 SUPPLIER
// 位置 + 每仓库一条 INTERNAL 位置）。
type StockLocation struct {
	ent.Schema
}

func (StockLocation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_locations",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存位置表"),
	}
}

func (StockLocation) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("位置名称").
			Optional().
			Nillable(),

		// parent_id / path：预留层级字段（初始实现扁平，值为 /{id}/）。
		field.Uint32("parent_id").
			Comment("父位置ID（预留层级，初始不用）").
			Default(0).
			Optional().
			Nillable(),

		field.String("path").
			Comment("物化路径（预留层级，初始为 /{id}/）").
			Optional().
			Nillable(),

		field.Enum("usage").
			Comment("位置用途：SUPPLIER=供应商（入库源），INTERNAL=内部仓库位置").
			NamedValues(
				"Supplier", "SUPPLIER",
				"Internal", "INTERNAL",
			).
			Default("INTERNAL").
			Optional().
			Nillable(),

		field.String("warehouse_code").
			Comment("归属仓库编码（仅 INTERNAL 位置有值）").
			Optional().
			Nillable(),
	}
}

func (StockLocation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockLocation) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 用途检索（如查租户的供应商位置）
		index.Fields("tenant_id", "usage").
			StorageKey("idx_inv_stock_location_tenant_usage"),

		// 按租户 + 仓库取该仓库的接收位置（INTERNAL）
		index.Fields("tenant_id", "warehouse_code").
			StorageKey("idx_inv_stock_location_tenant_warehouse"),
	}
}
