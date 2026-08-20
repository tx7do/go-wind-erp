package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Payable holds the schema definition for the Payable entity.
type Payable struct {
	ent.Schema
}

func (Payable) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_payables",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("应付单表"),
	}
}

// Fields of the Payable.
func (Payable) Fields() []ent.Field {
	return []ent.Field{
		field.String("payable_number").
			Comment("应付单号（服务端生成）").
			Optional().
			Nillable(),

		field.String("po_ref").
			Comment("来源采购单引用").
			Optional().
			Nillable(),

		field.String("supplier_code").
			Comment("供应商编码").
			Optional().
			Nillable(),

		field.Int64("amount").
			Comment("应付总额（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("paid_amount").
			Comment("已付金额（分，付款驱动）").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("应付状态").
			NamedValues(
				"Pending", "PENDING",
				"Partial", "PARTIAL",
				"Settled", "SETTLED",
				"Cancelled", "CANCELLED",
			).
			Default("PENDING").
			Optional().
			Nillable(),
	}
}

// Mixin of the Payable.
func (Payable) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Payable.
func (Payable) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 payable_number 唯一
		index.Fields("tenant_id", "payable_number").
			Unique().
			StorageKey("idx_fin_payable_tenant_number"),

		// 按租户 + 状态 + 创建时间，用于待付款列表
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_fin_payable_tenant_status_created_at"),

		// 按租户 + 供应商，用于供应商维度对账
		index.Fields("tenant_id", "supplier_code").
			StorageKey("idx_fin_payable_tenant_supplier"),
	}
}
