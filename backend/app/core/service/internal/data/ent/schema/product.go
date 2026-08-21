package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Product holds the schema definition for the Product entity.
type Product struct {
	ent.Schema
}

func (Product) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "prd_products",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("商品（SKU）主数据表"),
	}
}

// Fields of the Product.
func (Product) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("SKU编码").
			Optional().
			Nillable(),

		field.String("name").
			Comment("商品名称").
			Optional().
			Nillable(),

		field.String("spec").
			Comment("规格").
			Optional().
			Nillable(),

		field.String("unit").
			Comment("单位").
			Optional().
			Nillable(),

		field.Bool("enable").
			Comment("启用/禁用").
			Default(true).
			Optional().
			Nillable(),
	}
}

// Mixin of the Product.
func (Product) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Product.
func (Product) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_prd_product_tenant_code"),
	}
}
