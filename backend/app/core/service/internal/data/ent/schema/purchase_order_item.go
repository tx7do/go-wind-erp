package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// PurchaseOrderItem holds the schema definition for the PurchaseOrderItem entity.
type PurchaseOrderItem struct {
	ent.Schema
}

func (PurchaseOrderItem) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "pur_purchase_order_items",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("采购单明细表"),
	}
}

// Fields of the PurchaseOrderItem.
func (PurchaseOrderItem) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("po_id").
			Comment("所属采购单ID").
			Default(0).
			Optional().
			Nillable(),

		field.String("sku_code").
			Comment("SKU编码").
			Optional().
			Nillable(),

		field.Int64("quantity").
			Comment("采购数量").
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

		field.Int64("received_quantity").
			Comment("已收货数量").
			Default(0).
			Optional().
			Nillable(),
	}
}

// Mixin of the PurchaseOrderItem.
func (PurchaseOrderItem) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PurchaseOrderItem.
func (PurchaseOrderItem) Indexes() []ent.Index {
	return []ent.Index{
		// 按采购单取明细
		index.Fields("tenant_id", "po_id").
			StorageKey("idx_pur_po_item_tenant_po"),
	}
}
