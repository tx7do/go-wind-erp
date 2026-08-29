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

// Account 会计科目（简易总账）。平台级标准科目目录（tenant_id=0，查询走
// 系统视图——镜像 Plan/sys_permissions 的目录语义），服务启动时种子。
type Account struct {
	ent.Schema
}

func (Account) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "fin_accounts",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("会计科目表（平台标准目录）"),
	}
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("科目编码（如 1405 库存商品）").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("name").
			Comment("科目名称").
			NotEmpty().
			Optional().
			Nillable(),

		// category 报表归类：ASSET 资产 / LIABILITY 负债 / EQUITY 权益
		// / REVENUE 收入 / EXPENSE 费用。
		field.Enum("category").
			Comment("科目类别").
			NamedValues(
				"Asset", "ASSET",
				"Liability", "LIABILITY",
				"Equity", "EQUITY",
				"Revenue", "REVENUE",
				"Expense", "EXPENSE",
			).
			Default("ASSET").
			Optional().
			Nillable(),

		// balance_direction 余额方向：DEBIT 借方（资产/费用）/
		// CREDIT 贷方（负债/权益/收入）。
		field.Enum("balance_direction").
			Comment("余额方向").
			NamedValues(
				"Debit", "DEBIT",
				"Credit", "CREDIT",
			).
			Default("DEBIT").
			Optional().
			Nillable(),
	}
}

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").
			Unique().
			StorageKey("uix_fin_account_code"),
	}
}
