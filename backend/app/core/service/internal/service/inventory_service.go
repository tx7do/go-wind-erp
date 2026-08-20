package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// InventoryService 库存服务
type InventoryService struct {
	inventoryV1.UnimplementedInventoryServiceServer

	log *log.Helper

	inventoryRepo      *data.InventoryRepo
	warehouseRepo      *data.WarehouseRepo
	stockMovementRepo  *data.StockMovementRepo
}

func NewInventoryService(
	ctx *bootstrap.Context,
	inventoryRepo *data.InventoryRepo,
	warehouseRepo *data.WarehouseRepo,
	stockMovementRepo *data.StockMovementRepo,
) *InventoryService {
	svc := &InventoryService{
		log:               ctx.NewLoggerHelper("inventory/service/core-service"),
		inventoryRepo:     inventoryRepo,
		warehouseRepo:     warehouseRepo,
		stockMovementRepo: stockMovementRepo,
	}

	return svc
}

func (s *InventoryService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListInventoryResponse, error) {
	return s.inventoryRepo.List(ctx, req)
}

func (s *InventoryService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountInventoryResponse, error) {
	count, err := s.inventoryRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountInventoryResponse{Count: uint64(count)}, nil
}

// GetOverview 库存经营总览：聚合仓库数 / 在库 SKU 数 / 库存总量 / 流水数，
// 附按数量升序的低库存清单。读取走 TenantPrivacy 策略自动按调用者租户隔离。
func (s *InventoryService) GetOverview(ctx context.Context, req *inventoryV1.GetInventoryOverviewRequest) (*inventoryV1.InventoryOverview, error) {
	threshold := req.GetLowStockThreshold()
	if threshold <= 0 {
		threshold = 10
	}
	lowStockLimit := req.GetLowStockLimit()
	if lowStockLimit <= 0 {
		lowStockLimit = 10
	}

	warehouseCount, err := s.warehouseRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}

	skuCount, err := s.inventoryRepo.CountDistinctSku(ctx)
	if err != nil {
		return nil, err
	}

	totalQuantity, err := s.inventoryRepo.SumQuantity(ctx)
	if err != nil {
		return nil, err
	}

	movementCount, err := s.stockMovementRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}

	lowStockItems, err := s.inventoryRepo.ListLowStock(ctx, threshold, int(lowStockLimit))
	if err != nil {
		return nil, err
	}

	return &inventoryV1.InventoryOverview{
		WarehouseCount: uint64(warehouseCount),
		SkuCount:       uint64(skuCount),
		TotalQuantity:  totalQuantity,
		MovementCount:  uint64(movementCount),
		LowStockItems:  lowStockItems,
	}, nil
}

func (s *InventoryService) Get(ctx context.Context, req *inventoryV1.GetInventoryRequest) (*inventoryV1.Inventory, error) {
	return s.inventoryRepo.Get(ctx, req)
}

func (s *InventoryService) Create(ctx context.Context, req *inventoryV1.CreateInventoryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	// 新建库存记录的初始状态必须由状态机校验：AVAILABLE 是唯一合法的初始态。
	if req.Data.Status != nil && !validateStatusTransition(inventoryV1.Inventory_AVAILABLE, req.Data.GetStatus()) {
		return nil, inventoryV1.ErrorBadRequest("invalid initial inventory status")
	}

	if _, err := s.inventoryRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *InventoryService) Update(ctx context.Context, req *inventoryV1.UpdateInventoryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid parameter")
	}

	// 状态变更必须经状态机校验：先取当前状态，再校验 from→to 的转换是否被允许。
	if req.Data.Status != nil {
		old, err := s.inventoryRepo.Get(ctx, &inventoryV1.GetInventoryRequest{
			QueryBy: &inventoryV1.GetInventoryRequest_Id{Id: req.GetId()},
		})
		if err != nil {
			return nil, err
		}
		if !validateStatusTransition(old.GetStatus(), req.Data.GetStatus()) {
			return nil, inventoryV1.ErrorForbidden("unauthorized inventory status transition")
		}
	}

	if _, err := s.inventoryRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *InventoryService) Delete(ctx context.Context, req *inventoryV1.DeleteInventoryRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, inventoryV1.ErrorBadRequest("invalid request")
	}

	if err := s.inventoryRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
