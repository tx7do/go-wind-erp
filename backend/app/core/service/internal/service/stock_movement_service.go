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
)

// StockMovementService 库存流水服务
type StockMovementService struct {
	inventoryV1.UnimplementedStockMovementServiceServer

	log *log.Helper

	stockMovementRepo *data.StockMovementRepo
	inventoryRepo     *data.InventoryRepo
	purchaseOrderRepo *data.PurchaseOrderRepo
	approvalRequestRepo *data.ApprovalRequestRepo

	saga *procurementSagaSeam
}

func NewStockMovementService(
	ctx *bootstrap.Context,
	stockMovementRepo *data.StockMovementRepo,
	inventoryRepo *data.InventoryRepo,
	purchaseOrderRepo *data.PurchaseOrderRepo,
	approvalRequestRepo *data.ApprovalRequestRepo,
) *StockMovementService {
	svc := &StockMovementService{
		log:      ctx.NewLoggerHelper("stock_movement/service/core-service"),
		stockMovementRepo: stockMovementRepo,
	inventoryRepo:     inventoryRepo,
	purchaseOrderRepo: purchaseOrderRepo,
	approvalRequestRepo: approvalRequestRepo,

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
	if poID != 0 {
		if _, _, rerr := s.purchaseOrderRepo.ApplyReceipt(ctx, poID, skuCode, delta); rerr != nil {
			return nil, rerr
		}
	}

	// 原子回写库存（防负库存 + 防并发）；失败则冲正已累计的收货量
	// （单库内以反向补偿保持一致）。
	if _, err := s.inventoryRepo.ApplyDelta(ctx, inv.GetId(), delta); err != nil {
		if poID != 0 {
			if _, _, cerr := s.purchaseOrderRepo.ApplyReceipt(ctx, poID, skuCode, -delta); cerr != nil {
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

// Transfer 调拨：源仓出、目的仓入（两笔流水同备注关联）。入库失败时
// 以反向流水冲正源仓出库——SAGA 补偿语义的直接应用。
func (s *StockMovementService) Transfer(ctx context.Context, req *inventoryV1.TransferStockRequest) (*emptypb.Empty, error) {
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

	link := fmt.Sprintf("transfer:%s->%s:%s:%d", from, to, sku, time.Now().UnixMilli())
	remark := req.GetRemark()

	// 源仓出库。
	if _, err := s.Create(ctx, &inventoryV1.CreateStockMovementRequest{
		Data: &inventoryV1.StockMovement{
			WarehouseCode: trans.Ptr(from),
			SkuCode:       trans.Ptr(sku),
			Delta:         trans.Ptr(-qty),
			MovementType:  inventoryV1.StockMovement_TRANSFER.Enum(),
			Remark:        trans.Ptr(fmt.Sprintf("%s；out；%s", link, remark)),
		},
	}); err != nil {
		return nil, err
	}

	// 目的仓入库；失败则冲正源仓出库（补偿）。
	if _, err := s.Create(ctx, &inventoryV1.CreateStockMovementRequest{
		Data: &inventoryV1.StockMovement{
			WarehouseCode: trans.Ptr(to),
			SkuCode:       trans.Ptr(sku),
			Delta:         trans.Ptr(qty),
			MovementType:  inventoryV1.StockMovement_TRANSFER.Enum(),
			Remark:        trans.Ptr(fmt.Sprintf("%s；in；%s", link, remark)),
		},
	}); err != nil {
		s.log.Errorf("transfer inbound failed, compensating outbound: %s", err.Error())
		if _, cerr := s.Create(ctx, &inventoryV1.CreateStockMovementRequest{
			Data: &inventoryV1.StockMovement{
				WarehouseCode: trans.Ptr(from),
				SkuCode:       trans.Ptr(sku),
				Delta:         trans.Ptr(qty),
				MovementType:  inventoryV1.StockMovement_TRANSFER.Enum(),
				Remark:        trans.Ptr(fmt.Sprintf("%s；compensation；%s", link, remark)),
			},
		}); cerr != nil {
			s.log.Errorf("transfer compensation failed: %s", cerr.Error())
		}
		return nil, err
	}

	return &emptypb.Empty{}, nil
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
