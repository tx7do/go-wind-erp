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

// Plan 套餐（透明定价的全局目录，平台级 tenant_id=0）。
// 查询必须用系统视图（镜像 sys_permissions 的目录语义）——TenantPrivacy
// 会给租户上下文注入 tenant_id 过滤，直接查会得到空集。
type Plan struct {
	ent.Schema
}

func (Plan) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sub_plans",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("套餐表（透明定价目录）"),
	}
}

func (Plan) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("套餐编码：FREE/STANDARD/PRO").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("name").
			Comment("套餐名称").
			Optional().
			Nillable(),

		field.String("description").
			Comment("套餐说明（定价页展示）").
			Optional().
			Nillable(),

		field.Int64("price_cents").
			Comment("月价（分/月，FREE=0）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("max_users").
			Comment("用户数上限（0=无限）").
			Default(0).
			Optional().
			Nillable(),

		field.Int64("max_orders_monthly").
			Comment("月单量上限（PO+SO 合计，0=无限）").
			Default(0).
			Optional().
			Nillable(),
	}
}

func (Plan) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
		mixin.SortOrder{},
		mixin.SwitchStatus{},
	}
}

func (Plan) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").
			Unique().
			StorageKey("uix_sub_plan_code"),
	}
}
