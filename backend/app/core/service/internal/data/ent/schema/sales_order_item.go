package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SalesOrderItem holds the schema definition for the SalesOrderItem entity.
type SalesOrderItem struct {
	ent.Schema
}

func (SalesOrderItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sal_sales_order_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("销售单明细表"),
	}
}

func (SalesOrderItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("so_id").
			Comment("所属销售单ID").
			Default(0).
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU编码").
			Optional().
			Nillable(),

		field.Int64("quantity").
			Comment("销售数量").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("unit_price").
			Comment("单价（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("amount").
			Comment("明细金额（分，服务端计算）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("fulfilled_quantity").
			Comment("已履约数量").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (SalesOrderItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (SalesOrderItem) Indexes() []ent.Index {
	return []ent.Index{
		// 按销售单取明细
		index.Fields("tenant_id", "so_id").
			StorageKey("idx_sal_so_item_tenant_so"),
	}
}
