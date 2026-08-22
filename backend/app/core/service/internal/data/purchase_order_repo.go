package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"
	"github.com/tx7do/go-utils/trans"

	"go-wind-erp/app/core/service/internal/data/ent"
	"go-wind-erp/app/core/service/internal/data/ent/predicate"
	"go-wind-erp/app/core/service/internal/data/ent/purchaseorder"
	"go-wind-erp/app/core/service/internal/data/ent/purchaseorderitem"

	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

type PurchaseOrderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[procurementV1.PurchaseOrder, ent.PurchaseOrder]
	statusConverter *mapper.EnumTypeConverter[procurementV1.PurchaseOrder_Status, purchaseorder.Status]

	repository *entCrud.Repository[
		ent.PurchaseOrderQuery, ent.PurchaseOrderSelect,
		ent.PurchaseOrderCreate, ent.PurchaseOrderCreateBulk,
		ent.PurchaseOrderUpdate, ent.PurchaseOrderUpdateOne,
		ent.PurchaseOrderDelete,
		predicate.PurchaseOrder,
		procurementV1.PurchaseOrder, ent.PurchaseOrder,
	]
}

func NewPurchaseOrderRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *PurchaseOrderRepo {
	repo := &PurchaseOrderRepo{
		log:       ctx.NewLoggerHelper("purchase_order/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *PurchaseOrderRepo) init() {
	r.mapper = mapper.NewCopierMapper[procurementV1.PurchaseOrder, ent.PurchaseOrder]()
	r.statusConverter = mapper.NewEnumTypeConverter[procurementV1.PurchaseOrder_Status, purchaseorder.Status](procurementV1.PurchaseOrder_Status_name, procurementV1.PurchaseOrder_Status_value)

	r.repository = entCrud.NewRepository[
		ent.PurchaseOrderQuery, ent.PurchaseOrderSelect,
		ent.PurchaseOrderCreate, ent.PurchaseOrderCreateBulk,
		ent.PurchaseOrderUpdate, ent.PurchaseOrderUpdateOne,
		ent.PurchaseOrderDelete,
		predicate.PurchaseOrder,
		procurementV1.PurchaseOrder, ent.PurchaseOrder,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *PurchaseOrderRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().PurchaseOrder.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, procurementV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

// List 不携带明细（列表性能）；明细经 Get 获取。
func (r *PurchaseOrderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListPurchaseOrderResponse, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PurchaseOrder.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &procurementV1.ListPurchaseOrderResponse{Total: 0, Items: nil}, nil
	}

	return &procurementV1.ListPurchaseOrderResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// Get 携带明细组装。
func (r *PurchaseOrderRepo) Get(ctx context.Context, req *procurementV1.GetPurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	if req == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().PurchaseOrder.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *procurementV1.GetPurchaseOrderRequest_Id:
		whereCond = append(whereCond, purchaseorder.IDEQ(req.GetId()))
	}

	dto, err := r.repository.Get(ctx, builder, req.GetViewMask(), whereCond...)
	if err != nil {
		return nil, err
	}
	if dto == nil {
		return dto, nil
	}

	dto.Items = r.listItemDTOs(ctx, req.GetId())
	return dto, nil
}

// Create 落库采购单与明细（金额/总额由服务层预计算）。po_number 服务端生成。
func (r *PurchaseOrderRepo) Create(ctx context.Context, req *procurementV1.CreatePurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	poNumber := "PO" + fmt.Sprintf("%d", time.Now().UnixMilli())

	builder := r.entClient.Client().PurchaseOrder.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetPoNumber(poNumber).
		SetNillableSupplierCode(req.Data.SupplierCode).
		SetStatus(purchaseorder.StatusDraft).
		SetNillableTotalAmount(req.Data.TotalAmount).
		SetNillableWarehouseCode(req.Data.WarehouseCode).
		SetNillableRemark(req.Data.Remark).
		SetNillableCreatedBy(req.Data.CreatedBy).
		SetCreatedAt(time.Now())

	if req.Data.Id != nil {
		builder.SetID(req.GetData().GetId())
	}

	t, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert purchase_order failed: %s", err.Error())
		return nil, procurementV1.ErrorInternalServerError("insert purchase_order failed")
	}

	if err := r.replaceItems(ctx, t.TenantID, t.ID, req.Data.Items); err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(t)
	dto.Items = r.listItemDTOs(ctx, t.ID)
	return dto, nil
}

// Update 更新表头；当请求携带明细时整体替换（仅 DRAFT 允许，由服务层守卫）。
func (r *PurchaseOrderRepo) Update(ctx context.Context, req *procurementV1.UpdatePurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	if req == nil || req.Data == nil {
		return nil, procurementV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &procurementV1.CreatePurchaseOrderRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().PurchaseOrder.UpdateOneID(req.GetId())
	builder.Where(purchaseorder.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(purchaseorder.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *procurementV1.PurchaseOrder) {
			builder.
				SetNillableSupplierCode(req.Data.SupplierCode).
				SetNillableTotalAmount(req.Data.TotalAmount).
				SetNillableWarehouseCode(req.Data.WarehouseCode).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(purchaseorder.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return nil, err
	}

	if len(req.Data.Items) > 0 {
		poID := req.GetId()
		tid, _ := maybeTenantFromViewer(ctx)
		var tenantPtr *uint32
		if tid > 0 {
			tenantPtr = trans.Ptr(tid)
		} else if req.Data.TenantId != nil {
			tenantPtr = req.Data.TenantId
		}
		if err := r.replaceItems(ctx, tenantPtr, poID, req.Data.Items); err != nil {
			return nil, err
		}
		if result != nil {
			result.Items = r.listItemDTOs(ctx, poID)
		}
	}

	return result, nil
}

func (r *PurchaseOrderRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().PurchaseOrder.Query().
		Where(purchaseorder.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, procurementV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// TransitionStatus 原子状态迁移：仅当当前状态为 from 时才更新为 to。
// 0 行受影响说明已被并发变更，返回 Conflict（与审批/库存同模式）。
func (r *PurchaseOrderRepo) TransitionStatus(
	ctx context.Context,
	id uint32,
	from, to procurementV1.PurchaseOrder_Status,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().PurchaseOrder.Update().
		Where(purchaseorder.IDEQ(id)).
		Where(purchaseorder.StatusEQ(*r.statusConverter.ToEntity(trans.Ptr(from)))).
		SetStatus(*r.statusConverter.ToEntity(trans.Ptr(to))).
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(purchaseorder.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("transition purchase order status failed: %s", err.Error())
		return procurementV1.ErrorInternalServerError("transition purchase order status failed")
	}
	if n == 0 {
		return procurementV1.ErrorConflict("purchase order status changed concurrently")
	}
	return nil
}

// ApplyReceipt 收货回写：对匹配明细原子累计 received_quantity（条件
// received + qty <= ordered 防超收），随后若全部明细收齐且单据为
// APPROVED 则自动完结。返回 (本次收货后累计, 订单量, 是否触发自动完结)。
// autoCompleted 供调用方在整体成功后发下游通知。
func (r *PurchaseOrderRepo) ApplyReceipt(
	ctx context.Context,
	poID uint32,
	skuCode string,
	qty int64,
) (received int64, ordered int64, autoCompleted bool, err error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	item, err := r.entClient.Client().PurchaseOrderItem.Query().
		Where(purchaseorderitem.PoIDEQ(poID), purchaseorderitem.SkuCodeEQ(skuCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, 0, false, procurementV1.ErrorNotFound("purchase order item not found for sku")
		}
		return 0, 0, false, procurementV1.ErrorInternalServerError("query purchase order item failed")
	}

	// 收货闸门：仅 APPROVED 单据可收货，与付款轨对称。非 APPROVED
	// （DRAFT/SUBMITTED/REJECTED/CANCELLED/COMPLETED）一律拒绝，避免绕过
	// 审批链直接入库。以条件计数实现——PO 不存在或状态非 APPROVED 均记为
	// 冲突（计数为 0），消除状态枚举泄漏的同时关闭主要收货绕过路径。
	gateQuery := r.entClient.Client().PurchaseOrder.Query().
		Where(purchaseorder.IDEQ(poID)).
		Where(purchaseorder.StatusEQ(purchaseorder.StatusApproved))
	if hasTenant {
		gateQuery.Where(purchaseorder.TenantIDEQ(tid))
	}
	gateN, gerr := gateQuery.Count(ctx)
	if gerr != nil {
		r.log.Errorf("receiving status gate query failed: %s", gerr.Error())
		return 0, 0, false, procurementV1.ErrorInternalServerError("receiving status gate query failed")
	}
	if gateN != 1 {
		return 0, 0, false, procurementV1.ErrorConflict("purchase order is not approved for receiving")
	}

	ordered = *item.Quantity
	builder := r.entClient.Client().PurchaseOrderItem.Update().
		Where(purchaseorderitem.IDEQ(item.ID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d <= %s",
				s.C(purchaseorderitem.FieldReceivedQuantity), qty, s.C(purchaseorderitem.FieldQuantity),
			)))
		}).
		AddReceivedQuantity(qty)
	if hasTenant {
		builder.Where(purchaseorderitem.TenantIDEQ(tid))
	}

	n, serr := builder.Save(ctx)
	if serr != nil {
		r.log.Errorf("apply receipt failed: %s", serr.Error())
		return 0, 0, false, procurementV1.ErrorInternalServerError("apply receipt failed")
	}
	if n == 0 {
		return 0, 0, false, procurementV1.ErrorConflict("receipt exceeds ordered quantity")
	}

	received = *item.ReceivedQuantity + qty

	// 全部明细收齐且单据在途 → 自动完结（冲突/失败仅记录）。
	remaining, cerr := r.entClient.Client().PurchaseOrderItem.Query().
		Where(purchaseorderitem.PoIDEQ(poID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s < %s",
				s.C(purchaseorderitem.FieldReceivedQuantity), s.C(purchaseorderitem.FieldQuantity),
			)))
		}).
		Count(ctx)
	if cerr == nil && remaining == 0 {
		if terr := r.TransitionStatus(ctx, poID,
			procurementV1.PurchaseOrder_APPROVED, procurementV1.PurchaseOrder_COMPLETED); terr != nil {
			r.log.Warnf("auto-complete purchase order %d failed: %s", poID, terr.Error())
		} else {
			autoCompleted = true
		}
	}

	return received, ordered, autoCompleted, nil
}

// HasInFlightReplenishment 该 SKU 是否有在途补货：存在未收满的明细，
// 且其采购单处于 SUBMITTED/APPROVED。
func (r *PurchaseOrderRepo) HasInFlightReplenishment(ctx context.Context, skuCode string) (bool, error) {
	var ids []uint32
	if err := r.entClient.Client().PurchaseOrderItem.Query().
		Where(purchaseorderitem.SkuCodeEQ(skuCode)).
		Where(purchaseorderitem.PoIDNotNil()).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s < %s",
				s.C(purchaseorderitem.FieldReceivedQuantity), s.C(purchaseorderitem.FieldQuantity),
			)))
		}).
		GroupBy(purchaseorderitem.FieldPoID).
		Scan(ctx, &ids); err != nil {
		r.log.Errorf("query in-flight items failed: %s", err.Error())
		return false, procurementV1.ErrorInternalServerError("query in-flight items failed")
	}
	if len(ids) == 0 {
		return false, nil
	}

	return r.entClient.Client().PurchaseOrder.Query().
		Where(purchaseorder.IDIn(ids...)).
		Where(purchaseorder.StatusIn(purchaseorder.StatusSubmitted, purchaseorder.StatusApproved)).
		Exist(ctx)
}

// LastSupplierForSku 该 SKU 最近一次采购的供应商（无历史返回空串）。
func (r *PurchaseOrderRepo) LastSupplierForSku(ctx context.Context, skuCode string) (string, error) {
	var ids []uint32
	if err := r.entClient.Client().PurchaseOrderItem.Query().
		Where(purchaseorderitem.SkuCodeEQ(skuCode)).
		Where(purchaseorderitem.PoIDNotNil()).
		GroupBy(purchaseorderitem.FieldPoID).
		Scan(ctx, &ids); err != nil {
		return "", procurementV1.ErrorInternalServerError("query sku po history failed")
	}
	if len(ids) == 0 {
		return "", nil
	}

	last, err := r.entClient.Client().PurchaseOrder.Query().
		Where(purchaseorder.IDIn(ids...)).
		Order(ent.Desc(purchaseorder.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", nil
		}
		return "", procurementV1.ErrorInternalServerError("query last po failed")
	}
	return *last.SupplierCode, nil
}

// GetWarehouseCode 取采购单的收货仓库编码（PO 获批后据此确定 receiving location，
// 创建入库拣货单——Odoo PO→_create_picking 桥接的必要字段）。
func (r *PurchaseOrderRepo) GetWarehouseCode(ctx context.Context, poID uint32) (string, error) {
	po, err := r.entClient.Client().PurchaseOrder.Query().
		Where(purchaseorder.IDEQ(poID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", procurementV1.ErrorNotFound("purchase order not found")
		}
		return "", procurementV1.ErrorInternalServerError("query purchase order failed")
	}
	if po.WarehouseCode == nil {
		return "", procurementV1.ErrorBadRequest("purchase order has no warehouse_code")
	}
	return *po.WarehouseCode, nil
}

// GetItemsTx 取采购单明细（事务变体，供 StockPickingService.Validate 在事务内
// 累计 received_quantity 时使用）。
func (r *PurchaseOrderRepo) GetItemsTx(
	ctx context.Context,
	tx *ent.Tx,
	poID uint32,
) ([]*procurementV1.PurchaseOrderItem, error) {
	rows, err := tx.PurchaseOrderItem.Query().
		Where(purchaseorderitem.PoIDEQ(poID)).
		Order(ent.Asc(purchaseorderitem.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query purchase order items tx failed: %s", err.Error())
		return nil, procurementV1.ErrorInternalServerError("query purchase order items failed")
	}

	items := make([]*procurementV1.PurchaseOrderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &procurementV1.PurchaseOrderItem{
			Id:               trans.Ptr(row.ID),
			PoId:             row.PoID,
			SkuCode:          row.SkuCode,
			Quantity:         row.Quantity,
			UnitPrice:        row.UnitPrice,
			Amount:           row.Amount,
			ReceivedQuantity: row.ReceivedQuantity,
		})
	}
	return items, nil
}

// ApplyReceiptTx 与 ApplyReceipt 同语义的事务变体，接受显式 *ent.Tx，使
// StockPickingService.Validate 可在单个 DB 事务内运行（消除旧补偿竞态）。
func (r *PurchaseOrderRepo) ApplyReceiptTx(
	ctx context.Context,
	tx *ent.Tx,
	poItemID uint32,
	qty int64,
) (autoCompleted bool, err error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	item, err := tx.PurchaseOrderItem.Query().
		Where(purchaseorderitem.IDEQ(poItemID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, procurementV1.ErrorNotFound("purchase order item not found")
		}
		return false, procurementV1.ErrorInternalServerError("query purchase order item failed")
	}

	poID := *item.PoID

	// 收货闸门：仅 APPROVED 单据可收货（与 ApplyReceipt 对称）。
	gateQuery := tx.PurchaseOrder.Query().
		Where(purchaseorder.IDEQ(poID)).
		Where(purchaseorder.StatusEQ(purchaseorder.StatusApproved))
	if hasTenant {
		gateQuery.Where(purchaseorder.TenantIDEQ(tid))
	}
	gateN, gerr := gateQuery.Count(ctx)
	if gerr != nil {
		r.log.Errorf("receiving status gate query failed: %s", gerr.Error())
		return false, procurementV1.ErrorInternalServerError("receiving status gate query failed")
	}
	if gateN != 1 {
		return false, procurementV1.ErrorConflict("purchase order is not approved for receiving")
	}

	builder := tx.PurchaseOrderItem.Update().
		Where(purchaseorderitem.IDEQ(item.ID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d <= %s",
				s.C(purchaseorderitem.FieldReceivedQuantity), qty, s.C(purchaseorderitem.FieldQuantity),
			)))
		}).
		AddReceivedQuantity(qty)
	if hasTenant {
		builder.Where(purchaseorderitem.TenantIDEQ(tid))
	}

	n, serr := builder.Save(ctx)
	if serr != nil {
		r.log.Errorf("apply receipt tx failed: %s", serr.Error())
		return false, procurementV1.ErrorInternalServerError("apply receipt failed")
	}
	if n == 0 {
		return false, procurementV1.ErrorConflict("receipt exceeds ordered quantity")
	}

	// 全部明细收齐且单据在途 → 自动完结（冲突/失败仅记录）。
	remaining, cerr := tx.PurchaseOrderItem.Query().
		Where(purchaseorderitem.PoIDEQ(poID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s < %s",
				s.C(purchaseorderitem.FieldReceivedQuantity), s.C(purchaseorderitem.FieldQuantity),
			)))
		}).
		Count(ctx)
	if cerr == nil && remaining == 0 {
		// 事务内状态迁移（仅当当前为 APPROVED 时才更新为 COMPLETED）。
		transitionBuilder := tx.PurchaseOrder.Update().
			Where(purchaseorder.IDEQ(poID)).
			Where(purchaseorder.StatusEQ(purchaseorder.StatusApproved)).
			SetStatus(purchaseorder.StatusCompleted).
			SetUpdatedAt(time.Now())
		if hasTenant {
			transitionBuilder.Where(purchaseorder.TenantIDEQ(tid))
		}
		tn, terr := transitionBuilder.Save(ctx)
		if terr != nil {
			r.log.Warnf("auto-complete purchase order %d failed: %s", poID, terr.Error())
		} else if tn > 0 {
			autoCompleted = true
		}
	}

	return autoCompleted, nil
}

// Delete 删除采购单及其明细。
func (r *PurchaseOrderRepo) Delete(ctx context.Context, req *procurementV1.DeletePurchaseOrderRequest) error {
	if req == nil {
		return procurementV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().PurchaseOrder.Delete()
	delBuilder.Where(purchaseorder.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(purchaseorder.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return procurementV1.ErrorNotFound("purchase_order not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return procurementV1.ErrorInternalServerError("delete failed")
	}

	itemDel := r.entClient.Client().PurchaseOrderItem.Delete().
		Where(purchaseorderitem.PoIDEQ(req.GetId()))
	if hasTenant {
		itemDel.Where(purchaseorderitem.TenantIDEQ(tid))
	}
	if _, err := itemDel.Exec(ctx); err != nil {
		r.log.Errorf("delete purchase order items failed: %s", err.Error())
	}

	return nil
}

// replaceItems 整体替换明细（删旧插新）。金额字段由服务层预计算后传入。
func (r *PurchaseOrderRepo) replaceItems(
	ctx context.Context,
	tenantID *uint32,
	poID uint32,
	items []*procurementV1.PurchaseOrderItem,
) error {
	if poID == 0 {
		return nil
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	if hasTenant {
		tenantID = trans.Ptr(tid)
	}

	if _, err := r.entClient.Client().PurchaseOrderItem.Delete().
		Where(purchaseorderitem.PoIDEQ(poID)).
		Exec(ctx); err != nil {
		r.log.Errorf("clear purchase order items failed: %s", err.Error())
		return procurementV1.ErrorInternalServerError("clear purchase order items failed")
	}

	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		builder := r.entClient.Client().PurchaseOrderItem.Create().
			SetPoID(poID).
			SetNillableSkuCode(item.SkuCode).
			SetNillableQuantity(item.Quantity).
			SetNillableUnitPrice(item.UnitPrice).
			SetNillableAmount(item.Amount).
			SetNillableReceivedQuantity(item.ReceivedQuantity).
			SetNillableTenantID(tenantID)
		if _, err := builder.Save(ctx); err != nil {
			r.log.Errorf("insert purchase order item failed: %s", err.Error())
			return procurementV1.ErrorInternalServerError("insert purchase order item failed")
		}
	}

	return nil
}

// listItemDTOs 查询并映射某采购单的明细。
func (r *PurchaseOrderRepo) listItemDTOs(ctx context.Context, poID uint32) []*procurementV1.PurchaseOrderItem {
	rows, err := r.entClient.Client().PurchaseOrderItem.Query().
		Where(purchaseorderitem.PoIDEQ(poID)).
		Order(ent.Asc(purchaseorder.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query purchase order items failed: %s", err.Error())
		return nil
	}

	items := make([]*procurementV1.PurchaseOrderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &procurementV1.PurchaseOrderItem{
			Id:               trans.Ptr(row.ID),
			PoId:             row.PoID,
			SkuCode:          row.SkuCode,
			Quantity:         row.Quantity,
			UnitPrice:        row.UnitPrice,
			Amount:           row.Amount,
			ReceivedQuantity: row.ReceivedQuantity,
		})
	}
	return items
}
