package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
	appmixin "go-wind-erp/pkg/entgo/mixin"
)

// Receipt holds the schema definition for the Receipt entity.
type Receipt struct {
	ent.Schema
}

func (Receipt) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_receipts",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("收款表（append-only 台账）"),
	}
}

func (Receipt) Fields() []ent.Field {
	return []ent.Field{
		field.String("receipt_number").
			Comment("收款单号（服务端生成）").
			Optional().
			Nillable(),

		field.Uint32("receivable_id").
			Comment("所属应收单ID").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("amount").
			Comment("收款金额（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Enum("method").
			Comment("收款方式").
			NamedValues(
				"BankTransfer", "BANK_TRANSFER",
				"Cash", "CASH",
				"Check", "CHECK",
				"Other", "OTHER",
			).
			Default("BANK_TRANSFER").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("收款状态").
			NamedValues(
				"Pending", "PENDING",
				"Applied", "APPLIED",
				"Rejected", "REJECTED",
			).
			Default("PENDING").
			Optional().
			Nillable(),
	}
}

func (Receipt) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (Receipt) Indexes() []ent.Index {
	return []ent.Index{
		// 按应收单取收款流水
		index.Fields("tenant_id", "receivable_id", "created_at").
			StorageKey("idx_fin_receipt_tenant_receivable_created_at"),
	}
}
