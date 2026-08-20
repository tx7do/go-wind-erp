package service

import (
	"context"

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
	if movementType == inventoryV1.StockMovement_TRANSFER {
		return nil, inventoryV1.ErrorBadRequest("transfer movement is not supported yet")
	}

	poID := req.Data.GetPoId()
	if poID != 0 && movementType != inventoryV1.StockMovement_INBOUND {
		return nil, inventoryV1.ErrorBadRequest("po_id is only valid for inbound movements")
	}

	// 入库允许库存行不存在（自动建行）；出库/调整要求已存在。
	if movementType == inventoryV1.StockMovement_INBOUND {
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

	// 原子回写（防负库存 + 防并发）；随后以服务端计算的 qb/qa 落流水。
	if _, err := s.inventoryRepo.ApplyDelta(ctx, inv.GetId(), delta); err != nil {
		return nil, err
	}

	// 收货联动：超收由 ApplyReceipt 的条件更新守卫；失败不回滚库存
	// （入库已生效），记录并返回，由调用方决定是否冲正。
	if poID != 0 {
		if _, _, rerr := s.purchaseOrderRepo.ApplyReceipt(ctx, poID, skuCode, delta); rerr != nil {
			s.log.Errorf("apply receipt for po %d failed after inventory applied: %s", poID, rerr.Error())
			return nil, rerr
		}
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

func (s *StockMovementService) Delete(ctx context.Context, req *inventoryV1.DeleteStockMovementRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	if err := s.stockMovementRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
