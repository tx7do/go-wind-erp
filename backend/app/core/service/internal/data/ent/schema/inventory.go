package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Inventory holds the schema definition for the Inventory entity.
type Inventory struct {
	ent.Schema
}

func (Inventory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_inventories",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("库存表"),
	}
}

// Fields of the Inventory.
func (Inventory) Fields() []ent.Field {
	return []ent.Field{
		field.String("warehouse_code").
			Comment("所属仓库编码").
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU编码").
			Optional().
			Nillable(),

		field.Int64("quantity").
			Comment("库存数量").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("库存状态").
			NamedValues(
				"Available", "AVAILABLE",
				"Locked", "LOCKED",
				"Quarantined", "QUARANTINED",
			).
			Default("AVAILABLE").
			Optional().
			Nillable(),
	}
}

// Mixin of the Inventory.
func (Inventory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Inventory.
func (Inventory) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 (warehouse_code, sku_code) 唯一
		index.Fields("tenant_id", "warehouse_code", "sku_code").
			Unique().
			StorageKey("idx_inv_inventory_tenant_warehouse_sku"),
	}
}
