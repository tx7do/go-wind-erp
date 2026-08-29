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

// ApprovalRequest holds the schema definition for the ApprovalRequest entity.
type ApprovalRequest struct {
	ent.Schema
}

func (ApprovalRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "apr_approval_requests",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("审批请求表"),
	}
}

// Fields of the ApprovalRequest.
func (ApprovalRequest) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("审批标题").
			Optional().
			Nillable(),

		field.String("biz_type").
			Comment("业务类型").
			Optional().
			Nillable(),

		field.String("biz_ref").
			Comment("业务单据引用").
			Optional().
			Nillable(),

		field.String("summary").
			Comment("事由摘要").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("审批状态").
			NamedValues(
				"Pending", "PENDING",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
				"Cancelled", "CANCELLED",
			).
			Default("PENDING").
			Optional().
			Nillable(),

		field.Uint32("applicant_id").
			Comment("申请人用户ID").
			Default(0).
			Optional().
			Nillable(),

		field.Uint32("approver_id").
			Comment("审批人用户ID").
			Optional().
			Nillable(),

		field.String("comment").
			Comment("审批意见").
			Optional().
			Nillable(),

		// 多级审批：创建时从生效流程快照级数（在途单不受流程后续编辑影响）。
		// total_steps<=1 = 传统单级审批（无流程），行为与历史完全一致。
		field.Uint32("current_step").
			Comment("当前审批级（从 1 起）").
			Default(1).
			Optional().
			Nillable(),

		field.Uint32("total_steps").
			Comment("总级数（<=1 为单级审批）").
			Default(1).
			Optional().
			Nillable(),

		field.Uint32("flow_id").
			Comment("快照来源流程ID（0=无流程）").
			Default(0).
			Optional().
			Nillable(),
	}
}

// Mixin of the ApprovalRequest.
func (ApprovalRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the ApprovalRequest.
func (ApprovalRequest) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 状态 + 创建时间，用于待审批列表（时间列放末尾）
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_apr_request_tenant_status_created_at"),

		// 按租户 + 申请人 + 创建时间，用于“我发起的”列表
		index.Fields("tenant_id", "applicant_id", "created_at").
			StorageKey("idx_apr_request_tenant_applicant_created_at"),
	}
}
