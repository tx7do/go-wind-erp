package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// PurchaseOrder holds the schema definition for the PurchaseOrder entity.
type PurchaseOrder struct {
	ent.Schema
}

func (PurchaseOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "pur_purchase_orders",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("采购单表"),
	}
}

// Fields of the PurchaseOrder.
func (PurchaseOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("po_number").
			Comment("采购单号（服务端生成）").
			Optional().
			Nillable(),

		field.String("supplier_code").
			Comment("供应商编码").
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("采购单状态").
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
			Comment("采购总额（分，服务端按明细计算）").
			Default(0).
			Optional().
			Nillable(),

		// warehouse_code：收货仓库。PO 获批后据此确定 receiving location，
		// 创建入库拣货单（Odoo PO→_create_picking 桥接的必要字段）。
		field.String("warehouse_code").
			Comment("收货仓库编码").
			Optional().
			Nillable(),
	}
}

// Mixin of the PurchaseOrder.
func (PurchaseOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PurchaseOrder.
func (PurchaseOrder) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 po_number 唯一
		index.Fields("tenant_id", "po_number").
			Unique().
			StorageKey("idx_pur_po_tenant_po_number"),

		// 按租户 + 状态 + 创建时间，用于待审批列表
		index.Fields("tenant_id", "status", "created_at").
			StorageKey("idx_pur_po_tenant_status_created_at"),

		// 按租户 + 创建人 + 创建时间，用于"我发起的"
		index.Fields("tenant_id", "created_by", "created_at").
			StorageKey("idx_pur_po_tenant_creator_created_at"),
	}
}
