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
	"go-wind-erp/app/core/service/internal/data/ent/salesorder"
	"go-wind-erp/app/core/service/internal/data/ent/salesorderitem"

	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
)

type SalesOrderRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper          *mapper.CopierMapper[salesV1.SalesOrder, ent.SalesOrder]
	statusConverter *mapper.EnumTypeConverter[salesV1.SalesOrder_Status, salesorder.Status]

	repository *entCrud.Repository[
		ent.SalesOrderQuery, ent.SalesOrderSelect,
		ent.SalesOrderCreate, ent.SalesOrderCreateBulk,
		ent.SalesOrderUpdate, ent.SalesOrderUpdateOne,
		ent.SalesOrderDelete,
		predicate.SalesOrder,
		salesV1.SalesOrder, ent.SalesOrder,
	]
}

func NewSalesOrderRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) *SalesOrderRepo {
	repo := &SalesOrderRepo{
		log:       ctx.NewLoggerHelper("sales_order/repo/core-service"),
		entClient: entClient,
	}

	repo.init()

	return repo
}

func (r *SalesOrderRepo) init() {
	r.mapper = mapper.NewCopierMapper[salesV1.SalesOrder, ent.SalesOrder]()
	r.statusConverter = mapper.NewEnumTypeConverter[salesV1.SalesOrder_Status, salesorder.Status](salesV1.SalesOrder_Status_name, salesV1.SalesOrder_Status_value)

	r.repository = entCrud.NewRepository[
		ent.SalesOrderQuery, ent.SalesOrderSelect,
		ent.SalesOrderCreate, ent.SalesOrderCreateBulk,
		ent.SalesOrderUpdate, ent.SalesOrderUpdateOne,
		ent.SalesOrderDelete,
		predicate.SalesOrder,
		salesV1.SalesOrder, ent.SalesOrder,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

func (r *SalesOrderRepo) Count(ctx context.Context, whereCond []func(s *sql.Selector)) (int, error) {
	builder := r.entClient.Client().SalesOrder.Query()
	if len(whereCond) != 0 {
		builder.Modify(whereCond...)
	}

	count, err := builder.Count(ctx)
	if err != nil {
		r.log.Errorf("query count failed: %s", err.Error())
		return 0, salesV1.ErrorInternalServerError("query count failed")
	}

	return count, nil
}

// List 不携带明细（列表性能）；明细经 Get 获取。
func (r *SalesOrderRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*salesV1.ListSalesOrderResponse, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SalesOrder.Query()

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &salesV1.ListSalesOrderResponse{Total: 0, Items: nil}, nil
	}

	return &salesV1.ListSalesOrderResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// Get 携带明细组装。
func (r *SalesOrderRepo) Get(ctx context.Context, req *salesV1.GetSalesOrderRequest) (*salesV1.SalesOrder, error) {
	if req == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().SalesOrder.Query()

	var whereCond []func(s *sql.Selector)
	switch req.QueryBy.(type) {
	default:
	case *salesV1.GetSalesOrderRequest_Id:
		whereCond = append(whereCond, salesorder.IDEQ(req.GetId()))
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

// Create 落库销售单与明细（金额/总额由服务层预计算）。so_number 服务端生成。
func (r *SalesOrderRepo) Create(ctx context.Context, req *salesV1.CreateSalesOrderRequest) (*salesV1.SalesOrder, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	soNumber := "SO" + fmt.Sprintf("%d", time.Now().UnixMilli())

	builder := r.entClient.Client().SalesOrder.Create().
		SetNillableTenantID(req.Data.TenantId).
		SetSoNumber(soNumber).
		SetNillableCustomerCode(req.Data.CustomerCode).
		SetStatus(salesorder.StatusDraft).
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
		r.log.Errorf("insert sales_order failed: %s", err.Error())
		return nil, salesV1.ErrorInternalServerError("insert sales_order failed")
	}

	if err := r.replaceItems(ctx, t.TenantID, t.ID, req.Data.Items); err != nil {
		return nil, err
	}

	dto := r.mapper.ToDTO(t)
	dto.Items = r.listItemDTOs(ctx, t.ID)
	return dto, nil
}

// Update 更新表头；当请求携带明细时整体替换（仅 DRAFT 允许，由服务层守卫）。
func (r *SalesOrderRepo) Update(ctx context.Context, req *salesV1.UpdateSalesOrderRequest) (*salesV1.SalesOrder, error) {
	if req == nil || req.Data == nil {
		return nil, salesV1.ErrorBadRequest("invalid parameter")
	}

	if req.GetAllowMissing() {
		exist, err := r.IsExist(ctx, req.GetId())
		if err != nil {
			return nil, err
		}
		if !exist {
			createReq := &salesV1.CreateSalesOrderRequest{Data: req.Data}
			createReq.Data.CreatedBy = createReq.Data.UpdatedBy
			createReq.Data.UpdatedBy = nil
			return r.Create(ctx, createReq)
		}
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)
	builder := r.entClient.Client().Debug().SalesOrder.UpdateOneID(req.GetId())
	builder.Where(salesorder.IDEQ(req.GetId()))
	if hasTenant {
		builder.Where(salesorder.TenantIDEQ(tid))
	}
	result, err := r.repository.UpdateOne(ctx, builder, req.Data, req.GetUpdateMask(),
		func(dto *salesV1.SalesOrder) {
			builder.
				SetNillableCustomerCode(req.Data.CustomerCode).
				SetNillableTotalAmount(req.Data.TotalAmount).
				SetNillableWarehouseCode(req.Data.WarehouseCode).
				SetNillableRemark(req.Data.Remark).
				SetUpdatedAt(time.Now())

			if hasUser {
				builder.SetUpdatedBy(callerUserID)
			}
		},
		func(s *sql.Selector) {
			s.Where(sql.EQ(salesorder.FieldID, req.GetId()))
		},
	)
	if err != nil {
		return nil, err
	}

	if len(req.Data.Items) > 0 {
		soID := req.GetId()
		tid, _ := maybeTenantFromViewer(ctx)
		var tenantPtr *uint32
		if tid > 0 {
			tenantPtr = trans.Ptr(tid)
		} else if req.Data.TenantId != nil {
			tenantPtr = req.Data.TenantId
		}
		if err := r.replaceItems(ctx, tenantPtr, soID, req.Data.Items); err != nil {
			return nil, err
		}
		if result != nil {
			result.Items = r.listItemDTOs(ctx, soID)
		}
	}

	return result, nil
}

func (r *SalesOrderRepo) IsExist(ctx context.Context, id uint32) (bool, error) {
	exist, err := r.entClient.Client().SalesOrder.Query().
		Where(salesorder.IDEQ(id)).
		Exist(ctx)
	if err != nil {
		r.log.Errorf("query exist failed: %s", err.Error())
		return false, salesV1.ErrorInternalServerError("query exist failed")
	}
	return exist, nil
}

// TransitionStatus 原子状态迁移：仅当当前状态为 from 时才更新为 to。
// 0 行受影响说明已被并发变更，返回 Conflict。
func (r *SalesOrderRepo) TransitionStatus(
	ctx context.Context,
	id uint32,
	from, to salesV1.SalesOrder_Status,
) error {
	tid, hasTenant := maybeTenantFromViewer(ctx)
	callerUserID, hasUser := viewerUserIDFromContext(ctx)

	builder := r.entClient.Client().SalesOrder.Update().
		Where(salesorder.IDEQ(id)).
		Where(salesorder.StatusEQ(*r.statusConverter.ToEntity(trans.Ptr(from)))).
		SetStatus(*r.statusConverter.ToEntity(trans.Ptr(to))).
		SetUpdatedAt(time.Now())
	if hasTenant {
		builder.Where(salesorder.TenantIDEQ(tid))
	}
	if hasUser {
		builder.SetUpdatedBy(callerUserID)
	}

	n, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("transition sales order status failed: %s", err.Error())
		return salesV1.ErrorInternalServerError("transition sales order status failed")
	}
	if n == 0 {
		return salesV1.ErrorConflict("sales order status changed concurrently")
	}
	return nil
}

// GetItemsTx 取销售单明细（事务变体，供 StockPickingService.Validate 在事务内
// 累计 fulfilled_quantity 时使用）。
func (r *SalesOrderRepo) GetItemsTx(
	ctx context.Context,
	tx *ent.Tx,
	soID uint32,
) ([]*salesV1.SalesOrderItem, error) {
	rows, err := tx.SalesOrderItem.Query().
		Where(salesorderitem.SoIDEQ(soID)).
		Order(ent.Asc(salesorderitem.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query sales order items tx failed: %s", err.Error())
		return nil, salesV1.ErrorInternalServerError("query sales order items failed")
	}

	items := make([]*salesV1.SalesOrderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &salesV1.SalesOrderItem{
			Id:                 trans.Ptr(row.ID),
			SoId:               row.SoID,
			SkuCode:            row.SkuCode,
			Quantity:           row.Quantity,
			UnitPrice:          row.UnitPrice,
			Amount:             row.Amount,
			FulfilledQuantity:  row.FulfilledQuantity,
		})
	}
	return items, nil
}

func (r *SalesOrderRepo) replaceItems(
	ctx context.Context,
	tenantID *uint32,
	soID uint32,
	items []*salesV1.SalesOrderItem,
) error {
	if soID == 0 {
		return nil
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	if hasTenant {
		tenantID = trans.Ptr(tid)
	}

	if _, err := r.entClient.Client().SalesOrderItem.Delete().
		Where(salesorderitem.SoIDEQ(soID)).
		Exec(ctx); err != nil {
		r.log.Errorf("clear sales order items failed: %s", err.Error())
		return salesV1.ErrorInternalServerError("clear sales order items failed")
	}

	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if item == nil {
			continue
		}
		builder := r.entClient.Client().SalesOrderItem.Create().
			SetSoID(soID).
			SetNillableSkuCode(item.SkuCode).
			SetNillableQuantity(item.Quantity).
			SetNillableUnitPrice(item.UnitPrice).
			SetNillableAmount(item.Amount).
			SetNillableFulfilledQuantity(item.FulfilledQuantity).
			SetNillableTenantID(tenantID)
		if _, err := builder.Save(ctx); err != nil {
			r.log.Errorf("insert sales order item failed: %s", err.Error())
			return salesV1.ErrorInternalServerError("insert sales order item failed")
		}
	}

	return nil
}

// listItemDTOs 查询并映射某销售单的明细。
func (r *SalesOrderRepo) listItemDTOs(ctx context.Context, soID uint32) []*salesV1.SalesOrderItem {
	rows, err := r.entClient.Client().SalesOrderItem.Query().
		Where(salesorderitem.SoIDEQ(soID)).
		Order(ent.Asc(salesorderitem.FieldID)).
		All(ctx)
	if err != nil {
		r.log.Errorf("query sales order items failed: %s", err.Error())
		return nil
	}

	items := make([]*salesV1.SalesOrderItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &salesV1.SalesOrderItem{
			Id:                 trans.Ptr(row.ID),
			SoId:               row.SoID,
			SkuCode:            row.SkuCode,
			Quantity:           row.Quantity,
			UnitPrice:          row.UnitPrice,
			Amount:             row.Amount,
			FulfilledQuantity:  row.FulfilledQuantity,
		})
	}
	return items
}

// ApplyFulfillmentTx 出库履约回写：对匹配明细原子累计 fulfilled_quantity
// （条件 fulfilled + qty <= ordered 防超履约），随后若全部明细履约齐且单据为
// APPROVED 则自动完结。镜像 PurchaseOrderRepo.ApplyReceiptTx。
func (r *SalesOrderRepo) ApplyFulfillmentTx(
	ctx context.Context,
	tx *ent.Tx,
	soItemID uint32,
	qty int64,
) (autoCompleted bool, err error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	item, err := tx.SalesOrderItem.Query().
		Where(salesorderitem.IDEQ(soItemID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, salesV1.ErrorNotFound("sales order item not found")
		}
		return false, salesV1.ErrorInternalServerError("query sales order item failed")
	}

	soID := *item.SoID

	// 履约闸门：仅 APPROVED 单据可履约。
	gateQuery := tx.SalesOrder.Query().
		Where(salesorder.IDEQ(soID)).
		Where(salesorder.StatusEQ(salesorder.StatusApproved))
	if hasTenant {
		gateQuery.Where(salesorder.TenantIDEQ(tid))
	}
	gateN, gerr := gateQuery.Count(ctx)
	if gerr != nil {
		r.log.Errorf("fulfillment status gate query failed: %s", gerr.Error())
		return false, salesV1.ErrorInternalServerError("fulfillment status gate query failed")
	}
	if gateN != 1 {
		return false, salesV1.ErrorConflict("sales order is not approved for fulfillment")
	}

	builder := tx.SalesOrderItem.Update().
		Where(salesorderitem.IDEQ(item.ID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s + %d <= %s",
				s.C(salesorderitem.FieldFulfilledQuantity), qty, s.C(salesorderitem.FieldQuantity),
			)))
		}).
		AddFulfilledQuantity(qty)
	if hasTenant {
		builder.Where(salesorderitem.TenantIDEQ(tid))
	}

	n, serr := builder.Save(ctx)
	if serr != nil {
		r.log.Errorf("apply fulfillment tx failed: %s", serr.Error())
		return false, salesV1.ErrorInternalServerError("apply fulfillment failed")
	}
	if n == 0 {
		return false, salesV1.ErrorConflict("fulfillment exceeds ordered quantity")
	}

	// 全部明细履约齐且单据在途 → 自动完结（冲突/失败仅记录）。
	remaining, cerr := tx.SalesOrderItem.Query().
		Where(salesorderitem.SoIDEQ(soID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s < %s",
				s.C(salesorderitem.FieldFulfilledQuantity), s.C(salesorderitem.FieldQuantity),
			)))
		}).
		Count(ctx)
	if cerr == nil && remaining == 0 {
		// 事务内状态迁移（仅当当前为 APPROVED 时才更新为 COMPLETED）。
		transitionBuilder := tx.SalesOrder.Update().
			Where(salesorder.IDEQ(soID)).
			Where(salesorder.StatusEQ(salesorder.StatusApproved)).
			SetStatus(salesorder.StatusCompleted).
			SetUpdatedAt(time.Now())
		if hasTenant {
			transitionBuilder.Where(salesorder.TenantIDEQ(tid))
		}
		tn, terr := transitionBuilder.Save(ctx)
		if terr != nil {
			r.log.Warnf("auto-complete sales order %d failed: %s", soID, terr.Error())
		} else if tn > 0 {
			autoCompleted = true
		}
	}

	return autoCompleted, nil
}

// ApplyFulfillmentReturnTx 销售退货负向回写：fulfilled_quantity -= qty。
// 状态门放宽为 APPROVED|COMPLETED（已完结单允许退货）；原子守卫
// fulfilled − qty ≥ 0 防超退；减后未履约齐且单据 COMPLETED → 重开为 APPROVED。
// 返回 (soID, unitPrice) 供调用方计算应收冲抵额（qty × unitPrice）。
func (r *SalesOrderRepo) ApplyFulfillmentReturnTx(
	ctx context.Context,
	tx *ent.Tx,
	soItemID uint32,
	qty int64,
) (uint32, int64, error) {
	tid, hasTenant := maybeTenantFromViewer(ctx)

	item, err := tx.SalesOrderItem.Query().
		Where(salesorderitem.IDEQ(soItemID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, 0, salesV1.ErrorNotFound("sales order item not found")
		}
		return 0, 0, salesV1.ErrorInternalServerError("query sales order item failed")
	}

	soID := *item.SoID
	var unitPrice int64
	if item.UnitPrice != nil {
		unitPrice = *item.UnitPrice
	}

	// 退货闸门：APPROVED（在途）或 COMPLETED（已完结可退）。
	gateQuery := tx.SalesOrder.Query().
		Where(salesorder.IDEQ(soID)).
		Where(salesorder.StatusIn(salesorder.StatusApproved, salesorder.StatusCompleted))
	if hasTenant {
		gateQuery.Where(salesorder.TenantIDEQ(tid))
	}
	gateN, gerr := gateQuery.Count(ctx)
	if gerr != nil {
		r.log.Errorf("return status gate query failed: %s", gerr.Error())
		return 0, 0, salesV1.ErrorInternalServerError("return status gate query failed")
	}
	if gateN != 1 {
		return 0, 0, salesV1.ErrorConflict("sales order is not returnable (must be approved or completed)")
	}

	builder := tx.SalesOrderItem.Update().
		Where(salesorderitem.IDEQ(item.ID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s - %d >= 0",
				s.C(salesorderitem.FieldFulfilledQuantity), qty,
			)))
		}).
		AddFulfilledQuantity(-qty)
	if hasTenant {
		builder.Where(salesorderitem.TenantIDEQ(tid))
	}

	n, serr := builder.Save(ctx)
	if serr != nil {
		r.log.Errorf("apply fulfillment return tx failed: %s", serr.Error())
		return 0, 0, salesV1.ErrorInternalServerError("apply fulfillment return failed")
	}
	if n == 0 {
		return 0, 0, salesV1.ErrorConflict("return exceeds fulfilled quantity")
	}

	// 减后未履约齐且单据已完结 → 重开为 APPROVED（允许继续发货）。
	remaining, cerr := tx.SalesOrderItem.Query().
		Where(salesorderitem.SoIDEQ(soID)).
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP(fmt.Sprintf(
				"%s < %s",
				s.C(salesorderitem.FieldFulfilledQuantity), s.C(salesorderitem.FieldQuantity),
			)))
		}).
		Count(ctx)
	if cerr == nil && remaining > 0 {
		reopenBuilder := tx.SalesOrder.Update().
			Where(salesorder.IDEQ(soID)).
			Where(salesorder.StatusEQ(salesorder.StatusCompleted)).
			SetStatus(salesorder.StatusApproved).
			SetUpdatedAt(time.Now())
		if hasTenant {
			reopenBuilder.Where(salesorder.TenantIDEQ(tid))
		}
		if _, terr := reopenBuilder.Save(ctx); terr != nil {
			r.log.Warnf("reopen sales order %d after return failed: %s", soID, terr.Error())
		}
	}

	return soID, unitPrice, nil
}

// RevenueByMonth 按月汇总 COMPLETED 销售单的 total_amount。SQL 端按
// created_at（精确时间戳）分组并 SUM(total_amount)，Go 侧将时间戳格式化
// 为 YYYY-MM 做月度分桶。GROUP BY created_at 不使用任何日期函数，
// 保证 PostgreSQL 兼容。
type RevenueRow struct {
	Month string
	Total int64
}

func (r *SalesOrderRepo) RevenueByMonth(ctx context.Context) ([]RevenueRow, error) {
	type rawRow struct {
		CreatedAt *time.Time `sql:"created_at"`
		Total     int64      `sql:"total"`
	}
	var rows []rawRow
	tid, hasTenant := maybeTenantFromViewer(ctx)

	builder := r.entClient.Client().SalesOrder.Query().
		Where(salesorder.StatusEQ(salesorder.StatusCompleted))
	if hasTenant {
		builder.Where(salesorder.TenantIDEQ(tid))
	}
	if err := builder.GroupBy(salesorder.FieldCreatedAt).
		Aggregate(
			ent.As(ent.Sum(salesorder.FieldTotalAmount), "total"),
		).
		Scan(ctx, &rows); err != nil {
		r.log.Errorf("revenue by month query failed: %s", err.Error())
		return nil, salesV1.ErrorInternalServerError("revenue query failed")
	}

	out := make([]RevenueRow, 0, len(rows))
	for _, row := range rows {
		if row.CreatedAt == nil {
			continue
		}
		out = append(out, RevenueRow{
			Month: row.CreatedAt.Format("2006-01"),
			Total: row.Total,
		})
	}
	return out, nil
}

// CountTenantSince 计租户自某时刻起创建的单据数（计费守卫的月配额计量）。
func (r *SalesOrderRepo) CountTenantSince(ctx context.Context, tenantID uint32, since time.Time) (int, error) {
	return r.entClient.Client().SalesOrder.Query().
		Where(salesorder.TenantIDEQ(tenantID)).
		Where(salesorder.CreatedAtGTE(since)).
		Count(ctx)
}

func (r *SalesOrderRepo) Delete(ctx context.Context, req *salesV1.DeleteSalesOrderRequest) error {
	if req == nil {
		return salesV1.ErrorBadRequest("invalid parameter")
	}

	tid, hasTenant := maybeTenantFromViewer(ctx)
	delBuilder := r.entClient.Client().SalesOrder.Delete()
	delBuilder.Where(salesorder.IDEQ(req.GetId()))
	if hasTenant {
		delBuilder.Where(salesorder.TenantIDEQ(tid))
	}
	if _, err := delBuilder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return salesV1.ErrorNotFound("sales_order not found")
		}

		r.log.Errorf("delete one data failed: %s", err.Error())

		return salesV1.ErrorInternalServerError("delete failed")
	}

	itemDel := r.entClient.Client().SalesOrderItem.Delete().
		Where(salesorderitem.SoIDEQ(req.GetId()))
	if hasTenant {
		itemDel.Where(salesorderitem.TenantIDEQ(tid))
	}
	if _, err := itemDel.Exec(ctx); err != nil {
		r.log.Errorf("delete sales order items failed: %s", err.Error())
	}

	return nil
}



