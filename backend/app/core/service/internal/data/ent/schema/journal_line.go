package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// JournalLine 记账凭证行：一借一贷或多借多贷（借贷合计相等由仓储层
// PostTx 强制）。amount 单位分，与业务单据一致。
type JournalLine struct {
	ent.Schema
}

func (JournalLine) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_journal_lines",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("记账凭证行表"),
	}
}

func (JournalLine) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("entry_id").
			Comment("所属凭证ID").
			Default(0).
			Optional().
			Nillable(),

		field.String("account_code").
			Comment("科目编码").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("summary").
			Comment("行摘要").
			Optional().
			Nillable(),

		field.Int64("debit").
			Comment("借方金额（分）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("credit").
			Comment("贷方金额（分）").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (JournalLine) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

func (JournalLine) Indexes() []ent.Index {
	return []ent.Index{
		// 按凭证取行（凭证详情）
		index.Fields("tenant_id", "entry_id").
			StorageKey("idx_fin_journal_line_tenant_entry"),

		// 科目余额表聚合：按科目汇总借贷
		index.Fields("tenant_id", "account_code").
			StorageKey("idx_fin_journal_line_tenant_account"),
	}
}
