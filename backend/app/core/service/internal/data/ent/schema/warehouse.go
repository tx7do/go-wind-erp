package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Warehouse holds the schema definition for the Warehouse entity.
type Warehouse struct {
	ent.Schema
}

func (Warehouse) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_warehouses",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("仓库表"),
	}
}

// Fields of the Warehouse.
func (Warehouse) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("仓库编码").
			Optional().
			Nillable(),

		field.String("name").
			Comment("仓库名称").
			Optional().
			Nillable(),

		field.String("location").
			Comment("仓库地址").
			Optional().
			Nillable(),

		field.Bool("enable").
			Comment("启用/禁用仓库").
			Default(false).
			Optional().
			Nillable(),
	}
}

// Mixin of the Warehouse.
func (Warehouse) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Warehouse.
func (Warehouse) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_inv_warehouse_tenant_code"),
	}
}
