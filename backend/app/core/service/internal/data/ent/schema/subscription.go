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

// Subscription 租户订阅（每租户一条当前订阅，tenant_id 唯一）。
// 过期不删数据、不强制改 status——守卫按 period_end 动态判断（到期只读语义）。
type Subscription struct {
	ent.Schema
}

func (Subscription) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sub_subscriptions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("租户订阅表（透明定价）"),
	}
}

func (Subscription) Fields() []ent.Field {
	return []ent.Field{
		field.String("plan_code").
			Comment("当前套餐编码（FREE/STANDARD/PRO）").
			NotEmpty().
			Optional().
			Nillable(),

		field.Time("period_start").
			Comment("订阅周期起点").
			Optional().
			Nillable(),

		field.Time("period_end").
			Comment("订阅周期终点（nil=永久，FREE 无到期）").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("订阅状态（过期由守卫按 period_end 动态判断，status 仅记录最近一次操作）").
			NamedValues(
				"Active", "ACTIVE",
				"Expired", "EXPIRED",
				"Cancelled", "CANCELLED",
			).
			Default("ACTIVE").
			Optional().
			Nillable(),
	}
}

func (Subscription) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (Subscription) Indexes() []ent.Index {
	return []ent.Index{
		// 每租户一条当前订阅
		index.Fields("tenant_id").
			Unique().
			StorageKey("uix_sub_subscription_tenant"),

		index.Fields("tenant_id", "plan_code").
			StorageKey("idx_sub_subscription_tenant_plan"),
	}
}
