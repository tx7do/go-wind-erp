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

// ApprovalFlow 审批流模板（多级审批）：每租户每业务类型至多一条生效流程。
//
// 提交业务单（PO/SO/付款/收款）创建审批请求时按 (tenant, biz_type) 取
// 生效流程，将级数快照进请求（current_step/total_steps）——在途单不受
// 后续流程编辑影响。级定义在 ApprovalFlowStep（seq 1..N，角色制审批人：
// 持有该角色编码的用户即本级候选审批人）。
type ApprovalFlow struct {
	ent.Schema
}

func (ApprovalFlow) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "apr_approval_flows",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("审批流模板表（多级审批）"),
	}
}

func (ApprovalFlow) Fields() []ent.Field {
	return []ent.Field{
		field.String("biz_type").
			Comment("业务类型：PURCHASE_ORDER/SALES_ORDER/PAYMENT/RECEIPT").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("name").
			Comment("流程名称").
			NotEmpty().
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("启用状态：ON=生效（同业务类型唯一），OFF=停用").
			NamedValues(
				"On", "ON",
				"Off", "OFF",
			).
			Default("ON").
			Optional().
			Nillable(),
	}
}

func (ApprovalFlow) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (ApprovalFlow) Indexes() []ent.Index {
	return []ent.Index{
		// 每租户每业务类型至多一条流程（服务层保证同租户唯一）
		index.Fields("tenant_id", "biz_type").
			StorageKey("idx_apr_flow_tenant_biz_type"),
	}
}
