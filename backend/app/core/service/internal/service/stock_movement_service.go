package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// StockMovementService 库存流水服务
type StockMovementService struct {
	inventoryV1.UnimplementedStockMovementServiceServer

	log *log.Helper

	stockMovementRepo *data.StockMovementRepo
	inventoryRepo     *data.InventoryRepo
	purchaseOrderRepo *data.PurchaseOrderRepo
	approvalRequestRepo *data.ApprovalRequestRepo

	notifier *approvalNotifier

	saga *procurementSagaSeam
}

func NewStockMovementService(
	ctx *bootstrap.Context,
	stockMovementRepo *data.StockMovementRepo,
	inventoryRepo *data.InventoryRepo,
	purchaseOrderRepo *data.PurchaseOrderRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
	messageRepo *data.InternalMessageRepo,
	recipientRepo *data.InternalMessageRecipientRepo,
) *StockMovementService {
	svc := &StockMovementService{
		log:      ctx.NewLoggerHelper("stock_movement/service/core-service"),
		stockMovementRepo: stockMovementRepo,
		inventoryRepo:     inventoryRepo,
		purchaseOrderRepo: purchaseOrderRepo,
		approvalRequestRepo: approvalRequestRepo,

		notifier: &approvalNotifier{
			messageRepo:   messageRepo,
			recipientRepo: recipientRepo,
			log:           ctx.NewLoggerHelper("stock_movement/service/core-service"),
		},

		saga: &procurementSagaSeam{
			inventoryRepo:       inventoryRepo,
			purchaseOrderRepo:    purchaseOrderRepo,
			approvalRequestRepo:  approvalRequestRepo,
		},
	}

	return svc
}

func (s *StockMovementService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockMovementResponse, error) {
	return s.stockMovementRepo.List(ctx, req)
}

func (s *StockMovementService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountStockMovementResponse, error) {
	count, err := s.stockMovementRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountStockMovementResponse{Count: uint64(count)}, nil
}

func (s *StockMovementService) Get(ctx context.Context, req *inventoryV1.GetStockMovementRequest) (*inventoryV1.StockMovement, error) {
	return s.stockMovementRepo.Get(ctx, req)
}

func (s *StockMovementService) Create(ctx context.Context, req *inventoryV1.CreateStockMovementRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	warehouseCode := req.Data.GetWarehouseCode()
	skuCode := req.Data.GetSkuCode()
	if warehouseCode == "" || skuCode == "" {
		return nil, inventoryV1.ErrorBadRequest("warehouse_code and sku_code are required")
	}

	delta := req.Data.GetDelta()
	if delta == 0 {
		return nil, inventoryV1.ErrorBadRequest("delta must not be zero")
	}

	movementType := req.Data.GetMovementType()

	poID := req.Data.GetPoId()
	if poID != 0 && movementType != inventoryV1.StockMovement_INBOUND {
		return nil, inventoryV1.ErrorBadRequest("po_id is only valid for inbound movements")
	}

	// 正向变动（入库/调拨入/正调整）允许库存行不存在（自动建行）；
	// 负向变动要求已存在。
	if delta > 0 {
		if err := s.inventoryRepo.EnsureForInbound(ctx, warehouseCode, skuCode); err != nil {
			return nil, err
		}
	}

	inv, err := s.inventoryRepo.FindByWarehouseSku(ctx, warehouseCode, skuCode)
	if err != nil {
		return nil, err
	}

	qb := inv.GetQuantity()
	qa, overflow := addChecked(qb, delta)
	if overflow {
		return nil, inventoryV1.ErrorBadRequest("quantity arithmetic overflow")
	}

	// 收货联动先行：超收守卫必须发生在库存回写之前，否则被拒的流水
	// 会污染库存（E2E 实测暴露）。ApplyReceipt 自身条件更新防并发超收。
	receiptAutoCompleted := false
	if poID != 0 {
		var rerr error
		if _, _, receiptAutoCompleted, rerr = s.purchaseOrderRepo.ApplyReceipt(ctx, poID, skuCode, delta); rerr != nil {
			return nil, rerr
		}
	}

	// 原子回写库存（防负库存 + 防并发）；失败则冲正已累计的收货量
	// （单库内以反向补偿保持一致）。
	if _, err := s.inventoryRepo.ApplyDelta(ctx, inv.GetId(), delta); err != nil {
		if poID != 0 {
			if _, _, _, cerr := s.purchaseOrderRepo.ApplyReceipt(ctx, poID, skuCode, -delta); cerr != nil {
				s.log.Errorf("compensate receipt for po %d failed: %s", poID, cerr.Error())
			}
		}
		return nil, err
	}

	// SAGA：出库后联动采购（低库存补货评估）。
	if delta < 0 {
		if nerr := s.saga.notifyProcurement(ctx, stockEvent{
			warehouseCode: warehouseCode,
			skuCode:       skuCode,
			delta:         delta,
		}); nerr != nil {
			s.log.Errorf("saga notify procurement failed: %s", nerr.Error())
		}
	}

	movement := req.Data
	movement.QuantityBefore = trans.Ptr(qb)
	movement.QuantityAfter = trans.Ptr(qa)
	if _, err := s.stockMovementRepo.Create(ctx, &inventoryV1.CreateStockMovementRequest{Data: movement}); err != nil {
		return nil, err
	}

	// 下游事件通知：本次收货使全部明细收齐、PO 已自动完结时，告知采购单
	// 创建人（失败仅记录，不阻塞收货主流程）。
	if poID != 0 && receiptAutoCompleted {
		if po, gerr := s.purchaseOrderRepo.Get(ctx, &procurementV1.GetPurchaseOrderRequest{
			QueryBy: &procurementV1.GetPurchaseOrderRequest_Id{Id: poID},
		}); gerr == nil {
			if nerr := s.notifier.notifyPOAutoCompleted(ctx, po.GetPoNumber(), po.GetCreatedBy()); nerr != nil {
				s.log.Errorf("notify po %d auto-complete failed: %s", poID, nerr.Error())
			}
		} else {
			s.log.Errorf("load po %d for auto-complete notify failed: %s", poID, gerr.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// Reverse 冲正流水：对指定流水记一笔等量反向流水（append-only 台账的
// 补偿约定）。重复冲正同一笔被拒绝（查 remark 冲正标记）。冲正会连带
// 回写库存（Create 的既有语义），但不回写采购收货（收货冲正属业务
// 决策，走人工调整）。
func (s *StockMovementService) Reverse(ctx context.Context, req *inventoryV1.ReverseStockMovementRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	original, err := s.stockMovementRepo.Get(ctx, &inventoryV1.GetStockMovementRequest{
		QueryBy: &inventoryV1.GetStockMovementRequest_Id{Id: req.GetId()},
	})
	if err != nil {
		return nil, err
	}
	if original.GetDelta() == 0 {
		return nil, inventoryV1.ErrorBadRequest("original movement has zero delta")
	}

	// 幂等：已被冲正的流水不可再冲（冲正流水 remark 携带 reversal 标记）。
	marker := fmt.Sprintf("reversal-of:%d", req.GetId())
	if reversed, rerr := s.stockMovementRepo.HasReversalMarker(ctx,
		original.GetWarehouseCode(), original.GetSkuCode(), marker); rerr == nil && reversed {
		return nil, inventoryV1.ErrorConflict("movement already reversed")
	}

	reason := req.GetReason()
	_, err = s.Create(ctx, &inventoryV1.CreateStockMovementRequest{
		Data: &inventoryV1.StockMovement{
			WarehouseCode: original.WarehouseCode,
			SkuCode:       original.SkuCode,
			Delta:         trans.Ptr(-original.GetDelta()),
			MovementType:  original.MovementType,
			Remark:        trans.Ptr(fmt.Sprintf("%s；%s", marker, reason)),
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Transfer 调拨：源仓出、目的仓入（两笔流水同备注关联）。双腿在单个 DB
// 事务内执行——任一腿失败则整体回滚，避免源仓已扣而目的仓未加的
// 不一致窗口（旧实现为先出库后入库的 SAGA 补偿，补偿失败仅记日志，
// 存在可实现的不一致）。提交成功后若源仓因此低库存，再异步触发补货评估。
func (s *StockMovementService) Transfer(ctx context.Context, req *inventoryV1.TransferStockRequest) (ret *emptypb.Empty, err error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}
	from := req.GetFromWarehouseCode()
	to := req.GetToWarehouseCode()
	sku := req.GetSkuCode()
	qty := req.GetQuantity()
	if from == "" || to == "" || sku == "" {
		return nil, inventoryV1.ErrorBadRequest("from/to warehouse and sku are required")
	}
	if from == to {
		return nil, inventoryV1.ErrorBadRequest("from and to warehouse must differ")
	}
	if qty <= 0 {
		return nil, inventoryV1.ErrorBadRequest("quantity must be positive")
	}

	// 开启事务：双腿的库存回写与流水落库均在事务内进行；FinishTx 依命名
	// 返回值 err 决定提交或回滚（任一腿失败即回滚，无需补偿）。
	tx, terr := s.inventoryRepo.BeginTx(ctx)
	if terr != nil {
		return nil, terr
	}
	defer func() { s.inventoryRepo.FinishTx(tx, err) }()

	link := fmt.Sprintf("transfer:%s->%s:%s:%d", from, to, sku, time.Now().UnixMilli())
	remark := req.GetRemark()

	// 源仓出库（delta = -qty）。双腿的库存回写与流水落库均通过 *Tx 仓储
	// 方法在事务 tx 内执行（与 tenant/user 服务的 WithTx 模式一致）。
	// EnsureForInbound 仅对正向变动调用；出库为负向，故跳过。
	srcInv, serr := s.inventoryRepo.FindByWarehouseSkuTx(ctx, tx, from, sku)
	if serr != nil {
		err = serr
		return nil, err
	}
	srcQb := srcInv.GetQuantity()
	srcQa, srcOverflow := addChecked(srcQb, -qty)
	if srcOverflow {
		err = inventoryV1.ErrorBadRequest("quantity arithmetic overflow")
		return nil, err
	}
	if _, e := s.inventoryRepo.ApplyDeltaTx(ctx, tx, srcInv.GetId(), -qty); e != nil {
		err = e
		return nil, err
	}
	if _, e := s.stockMovementRepo.CreateTx(ctx, tx, &inventoryV1.CreateStockMovementRequest{
		Data: &inventoryV1.StockMovement{
			WarehouseCode:  trans.Ptr(from),
			SkuCode:        trans.Ptr(sku),
			Delta:          trans.Ptr(-qty),
			MovementType:   inventoryV1.StockMovement_TRANSFER.Enum(),
			Remark:         trans.Ptr(fmt.Sprintf("%s；out；%s", link, remark)),
			QuantityBefore: trans.Ptr(srcQb),
			QuantityAfter:  trans.Ptr(srcQa),
		},
	}); e != nil {
		err = e
		return nil, err
	}

	// 目的仓入库（delta = +qty）。正向变动先确保库存行存在。
	if e := s.inventoryRepo.EnsureForInboundTx(ctx, tx, to, sku); e != nil {
		err = e
		return nil, err
	}
	dstInv, derr := s.inventoryRepo.FindByWarehouseSkuTx(ctx, tx, to, sku)
	if derr != nil {
		err = derr
		return nil, err
	}
	dstQb := dstInv.GetQuantity()
	dstQa, dstOverflow := addChecked(dstQb, qty)
	if dstOverflow {
		err = inventoryV1.ErrorBadRequest("quantity arithmetic overflow")
		return nil, err
	}
	if _, e := s.inventoryRepo.ApplyDeltaTx(ctx, tx, dstInv.GetId(), qty); e != nil {
		err = e
		return nil, err
	}
	if _, e := s.stockMovementRepo.CreateTx(ctx, tx, &inventoryV1.CreateStockMovementRequest{
		Data: &inventoryV1.StockMovement{
			WarehouseCode:  trans.Ptr(to),
			SkuCode:        trans.Ptr(sku),
			Delta:          trans.Ptr(qty),
			MovementType:   inventoryV1.StockMovement_TRANSFER.Enum(),
			Remark:         trans.Ptr(fmt.Sprintf("%s；in；%s", link, remark)),
			QuantityBefore: trans.Ptr(dstQb),
			QuantityAfter:  trans.Ptr(dstQa),
		},
	}); e != nil {
		err = e
		return nil, err
	}

	// 两腿均成功 → 提交事务（由 FinishTx 执行）。
	ret = &emptypb.Empty{}
	err = nil

	// 事务提交后，源仓出库可能触发低库存 → 补货评估（与 Create 的
	// 出库语义一致，失败仅记日志）。
	if nerr := s.saga.notifyProcurement(ctx, stockEvent{
		warehouseCode: from,
		skuCode:       sku,
		delta:         -qty,
	}); nerr != nil {
		s.log.Errorf("saga notify procurement failed: %s", nerr.Error())
	}
	return ret, err
}


func (s *StockMovementService) Delete(ctx context.Context, req *inventoryV1.DeleteStockMovementRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	if err := s.stockMovementRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
