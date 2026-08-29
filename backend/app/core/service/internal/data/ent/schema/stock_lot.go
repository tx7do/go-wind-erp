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

// StockLot 批次登记（借鉴 Odoo stock.production.lot）。
//
// 记录式批次/效期管理：本表只登记批号与效期，批次剩余数量由
// inv_stock_move_lines 按位置 usage 聚合推导（入 INTERNAL 为 +、出
// INTERNAL 为 −），不参与 StockQuant 引擎——现有进销存闭环零改动。
// 批次在收货 Validate 时按 (sku, 批号) get-or-create；出库未指派批次时
// 服务端按 FEFO（效期升序）自动拆分扣减。
type StockLot struct {
	ent.Schema
}

func (StockLot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "inv_stock_lots",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("批次登记表（记录式批次/效期）"),
	}
}

func (StockLot) Fields() []ent.Field {
	return []ent.Field{
		field.String("sku_code").
			Comment("产品编码").
			NotEmpty().
			Optional().
			Nillable(),

		field.String("name").
			Comment("批次号").
			NotEmpty().
			Optional().
			Nillable(),

		field.Time("expiry_date").
			Comment("效期（可空=不限期）").
			Optional().
			Nillable(),
	}
}

func (StockLot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{}, appmixin.SoftDeleteQuery{},
		mixin.OperatorID{},
		mixin.Remark{},
		mixin.TenantID[uint32]{},
	}
}

func (StockLot) Indexes() []ent.Index {
	return []ent.Index{
		// 同租户同 SKU 批次号唯一（get-or-create 依据）
		index.Fields("tenant_id", "sku_code", "name").
			Unique().
			StorageKey("uix_inv_stock_lot_tenant_sku_name"),

		// FEFO 扫描：按 SKU 取效期升序的可扣批次
		index.Fields("tenant_id", "sku_code", "expiry_date").
			StorageKey("idx_inv_stock_lot_tenant_sku_expiry"),
	}
}
