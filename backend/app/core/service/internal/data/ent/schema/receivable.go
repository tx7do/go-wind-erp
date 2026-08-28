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

// Receivable holds the schema definition for the Receivable entity.
type Receivable struct {
	ent.Schema
}

func (Receivable) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_receivables",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("应收单表"),
	}
}

func (Receivable) Fields() []ent.Field {
	return []ent.Field{
		field.String("receivable_number").
			Comment("应收单号（服务端生成）").
			Optional().
			Nillable(),

		field.String("so_ref").
			Comment("来源销售单引用").
			Optional().
			Nillable(),

		field.String("customer_code").
			Comment("客户编码").
			Optional().
			Nillable(),

		field.Int64("amount").
			Comment("应收总额（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("paid_amount").
			Comment("已收金额（分，收款驱动）").
			Default(0).
			Optional().
			Nillable(),

		field.Time("due_date").
			Comment("账期到期日").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("应收状态").
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

func (Receivable) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (Receivable) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 receivable_number 唯一
		index.Fields("tenant_id", "receivable_number").
			Unique().
			StorageKey("idx_fin_receivable_tenant_number"),

		// 按租户 + 状态 + 创建时间，用于待收款列表
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_fin_receivable_tenant_status_created_at"),

		// 按租户 + 客户，用于客户维度对账
		index.Fields("tenant_id", "customer_code").
			StorageKey("idx_fin_receivable_tenant_customer"),
	}
}
