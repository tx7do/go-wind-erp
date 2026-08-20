package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// StockMovement holds the schema definition for the StockMovement entity.
type StockMovement struct {
	ent.Schema
}

func (StockMovement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_movements",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存流水表"),
	}
}

// Fields of theStockMovement.
func (StockMovement) Fields() []ent.Field {
	return []ent.Field{
		field.String("warehouse_code").
			Comment("涉及仓库编码").
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("涉及SKU编码").
			Optional().
			Nillable(),

		field.Int64("delta").
			Comment("数量变化量").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("movement_type").
			Comment("流水类型").
			NamedValues(
				"Inbound", "INBOUND",
				"Outbound", "OUTBOUND",
				"Transfer", "TRANSFER",
				"Adjustment", "ADJUSTMENT",
			).
			Default("INBOUND").
			Optional().
			Nillable(),

		field.Int64("quantity_before").
			Comment("操作前数量").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("quantity_after").
			Comment("操作后数量").
			Default(0).
			Optional().
			Nillable(),
	}
}

// Mixin of the StockMovement.
func (StockMovement) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the StockMovement.
func (StockMovement) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 创建时间，用于租户范围的时间区间查询与分页（时间列放末尾）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_inv_stock_movement_tenant_created_at"),

		// 按租户 + 仓库 + SKU，用于按仓库/SKU 检索流水
		index.Fields("tenant_id", "warehouse_code", "sku_code").
			StorageKey("idx_inv_stock_movement_tenant_warehouse_sku"),
	}
}
