package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Payment holds the schema definition for the Payment entity.
type Payment struct {
	ent.Schema
}

func (Payment) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_payments",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("付款表（append-only 台账）"),
	}
}

// Fields of the Payment.
func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.String("payment_number").
			Comment("付款单号（服务端生成）").
			Optional().
			Nillable(),

		field.Uint32("payable_id").
			Comment("所属应付单ID").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("amount").
			Comment("付款金额（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("method").
			Comment("付款方式").
			NamedValues(
				"BankTransfer", "BANK_TRANSFER",
				"Cash", "CASH",
				"Check", "CHECK",
				"Other", "OTHER",
			).
			Default("BANK_TRANSFER").
			Optional().
			Nillable(),
	}
}

// Mixin of the Payment.
func (Payment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Payment.
func (Payment) Indexes() []ent.Index {
	return []ent.Index{
		// 按应付单取付款流水
		index.Fields("tenant_id", "payable_id", "created_at").
			StorageKey("idx_fin_payment_tenant_payable_created_at"),
	}
}
