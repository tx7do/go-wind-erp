package service

import (
	"context"
	"strings"

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

	// AVAILABLE 是唯一合法的初始态（nil 缺省即 AVAILABLE）。不能用
	// validateStatusTransition 判断：AVAILABLE→LOCKED/QUARANTINED 本就是
	// 合法出边，那样校验形同虚设。
	if req.Data.GetStatus() != inventoryV1.Inventory_AVAILABLE {
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

	// 状态变更走原子条件迁移（仅 from 状态可迁），避免读-验-写的并发竞态；
	// 其余字段仍走常规 Update。业务拒绝用 409 Conflict（403+FORBIDDEN 会被
	// 移动端统一拦截器判为会话失效触发登出）。
	if req.Data.Status != nil {
		old, err := s.inventoryRepo.Get(ctx, &inventoryV1.GetInventoryRequest{
			QueryBy: &inventoryV1.GetInventoryRequest_Id{Id: req.GetId()},
		})
		if err != nil {
			// allow_missing + 记录不存在 → 转创建路径，同样受初始态约束。
			if req.GetAllowMissing() && isNotFoundError(err) {
				if req.Data.GetStatus() != inventoryV1.Inventory_AVAILABLE {
					return nil, inventoryV1.ErrorBadRequest("invalid initial inventory status")
				}
				req.Data.Status = nil
				if _, err := s.inventoryRepo.Update(ctx, req); err != nil {
					return nil, err
				}
				return &emptypb.Empty{}, nil
			}
			return nil, err
		}

		if !validateStatusTransition(old.GetStatus(), req.Data.GetStatus()) {
			return nil, inventoryV1.ErrorConflict("inventory status transition not allowed")
		}

		if err := s.inventoryRepo.TransitionStatus(ctx, req.GetId(), old.GetStatus(), req.Data.GetStatus()); err != nil {
			return nil, err
		}

		// 状态已迁移完成；剩余字段更新中剔除状态，避免二次写入引入竞态。
		req.Data.Status = nil
		req.UpdateMask.Paths = removeMaskPath(req.UpdateMask.GetPaths(), "status")
	}

	if _, err := s.inventoryRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// isNotFoundError 判断错误是否为资源不存在（ent NotFound 或 proto 404）。
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NOT_FOUND") {
		return true
	}
	return false
}

// removeMaskPath 从 FieldMask 路径中剔除指定字段。
func removeMaskPath(paths []string, target string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p != target {
			out = append(out, p)
		}
	}
	return out
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
