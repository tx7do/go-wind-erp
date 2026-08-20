package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Supplier holds the schema definition for the Supplier entity.
type Supplier struct {
	ent.Schema
}

func (Supplier) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "pur_suppliers",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("供应商表"),
	}
}

// Fields of the Supplier.
func (Supplier) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("供应商编码").
			Optional().
			Nillable(),

		field.String("name").
			Comment("供应商名称").
			Optional().
			Nillable(),

		field.String("contact").
			Comment("联系人").
			Optional().
			Nillable(),

		field.String("phone").
			Comment("联系电话").
			Optional().
			Nillable(),

		field.Bool("enable").
			Comment("启用/禁用供应商").
			Default(false).
			Optional().
			Nillable(),
	}
}

// Mixin of the Supplier.
func (Supplier) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Supplier.
func (Supplier) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_pur_supplier_tenant_code"),
	}
}
