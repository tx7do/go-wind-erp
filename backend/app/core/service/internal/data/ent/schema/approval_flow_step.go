package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// ApprovalFlowStep 审批级定义：seq 1..N 逐级推进，role_code 为审批人角色
// （角色目录 sys_roles.code，持有该角色的用户即本级候选审批人）。
// 级随流程整体替换（编辑流程=删全级重建），不独立软删。
type ApprovalFlowStep struct {
	ent.Schema
}

func (ApprovalFlowStep) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "apr_approval_flow_steps",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("审批级定义表（多级审批）"),
	}
}

func (ApprovalFlowStep) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("flow_id").
			Comment("所属流程ID").
			Default(0).
			Optional().
			Nillable(),

		field.Uint32("seq").
			Comment("级序（1..N，从 1 起）").
			Default(1).
			Optional().
			Nillable(),

		field.String("name").
			Comment("级名称（如：主管复核/经理终审）").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("role_code").
			Comment("本级审批人角色编码").
			NotEmpty().
			Optional().
			Nillable(),
	}
}

func (ApprovalFlowStep) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

func (ApprovalFlowStep) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "flow_id", "seq").
			StorageKey("idx_apr_flow_step_tenant_flow_seq"),
	}
}
