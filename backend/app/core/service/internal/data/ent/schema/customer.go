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

// Customer holds the schema definition for the Customer entity.
type Customer struct {
	ent.Schema
}

func (Customer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sal_customers",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("客户表"),
	}
}

func (Customer) Fields() []ent.Field {
	return []ent.Field{
		field.String("code").
			Comment("客户编码").
			Optional().
			Nillable(),

		field.String("name").
			Comment("客户名称").
			Optional().
			Nillable(),

		field.String("contact").
			Comment("联系人").
			Optional().
			Nillable(),

		field.String("phone").
			Comment("联系电话").
			Optional().
			Nillable(),

		// credit_limit：信用额度（分，0=不限额）。销售单审批时校验：
		// 该客户应收未清余额 + 本单金额 > 额度 → 拦截。
		field.Int64("credit_limit").
			Comment("信用额度（分，0=不限）").
			Default(0).
			Optional().
			Nillable(),

		field.Bool("enable").
			Comment("启用/禁用客户").
			Default(false).
			Optional().
			Nillable(),
	}
}

func (Customer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (Customer) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 code 唯一
		index.Fields("tenant_id", "code").
			Unique().
			StorageKey("idx_sal_customer_tenant_code"),
	}
}
