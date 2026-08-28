package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SalesOrder holds the schema definition for the SalesOrder entity.
type SalesOrder struct {
	ent.Schema
}

func (SalesOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "sal_sales_orders",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("销售单表"),
	}
}

func (SalesOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("so_number").
			Comment("销售单号（服务端生成）").
			Optional().
			Nillable(),

		field.String("customer_code").
			Comment("客户编码").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("销售单状态").
			NamedValues(
				"Draft", "DRAFT",
				"Submitted", "SUBMITTED",
				"Approved", "APPROVED",
				"Rejected", "REJECTED",
				"Completed", "COMPLETED",
				"Cancelled", "CANCELLED",
			).
			Default("DRAFT").
			Optional().
			Nillable(),

		field.Int64("total_amount").
			Comment("销售总额（分，服务端按明细计算）").
			Default(0).
			Optional().
			Nillable(),

		// warehouse_code：发货仓库。SO 获批后据此确定出库 source location，
		// 创建出库拣货单（镜像 PO 的 warehouse_code→receiving location 桥接）。
		field.String("warehouse_code").
			Comment("发货仓库编码").
			Optional().
			Nillable(),
	}
}

func (SalesOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (SalesOrder) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 so_number 唯一
		index.Fields("tenant_id", "so_number").
			Unique().
			StorageKey("idx_sal_so_tenant_so_number"),

		// 按租户 + 状态 + 创建时间，用于待审批列表
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_sal_so_tenant_status_created_at"),

		// 按租户 + 创建人 + 创建时间，用于"我发起的"
		index.Fields("tenant_id", "created_by", "created_at").
			StorageKey("idx_sal_so_tenant_creator_created_at"),
	}
}
