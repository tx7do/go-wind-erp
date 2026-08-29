package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// JournalEntry 记账凭证头（简易总账）。业务事件（收发货/审批/收付款）
// 在同一事务内生成的平衡分录；append-only 审计轨迹，无更新。
type JournalEntry struct {
	ent.Schema
}

func (JournalEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_journal_entries",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("记账凭证头表"),
	}
}

func (JournalEntry) Fields() []ent.Field {
	return []ent.Field{
		field.String("entry_number").
			Comment("凭证号（JE-<毫秒时间戳>）").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("summary").
			Comment("摘要（如：采购入库 PO123）").
			Optional().
			Nillable(),

		field.String("biz_ref").
			Comment("业务来源引用（如 STOCK_PICKING:17 / RECEIPT:5）").
			Optional().
			Nillable(),

		field.Time("entry_date").
			Comment("记账日期（业务发生时间）").
			Optional().
			Nillable(),
	}
}

func (JournalEntry) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.OperatorID{},
		mixin.TenantID[uint32]{},
	}
}

func (JournalEntry) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 日期倒序（凭证流水页）
		index.Fields("tenant_id", "entry_date").
			StorageKey("idx_fin_journal_entry_tenant_date"),

		// 按业务来源追溯（业务单据 → 凭证）
		index.Fields("tenant_id", "biz_ref").
			StorageKey("idx_fin_journal_entry_tenant_biz_ref"),
	}
}
