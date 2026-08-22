package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockQuantService 库存量服务（借鉴 Odoo stock.quant，只读——quantity 仅由
// 拣货校验变更）。提供看板聚合（GetOverview）与流水趋势（GetMovementTrend）。
type StockQuantService struct {
	inventoryV1.UnimplementedStockQuantServiceServer

	log *log.Helper

	stockQuantRepo    *data.StockQuantRepo
	stockMoveLineRepo *data.StockMoveLineRepo
	warehouseRepo     *data.WarehouseRepo
}

func NewStockQuantService(
	ctx *bootstrap.Context,
	stockQuantRepo *data.StockQuantRepo,
	stockMoveLineRepo *data.StockMoveLineRepo,
	warehouseRepo *data.WarehouseRepo,
) *StockQuantService {
	svc := &StockQuantService{
		log:               ctx.NewLoggerHelper("stock_quant/service/core-service"),
		stockQuantRepo:    stockQuantRepo,
		stockMoveLineRepo: stockMoveLineRepo,
		warehouseRepo:     warehouseRepo,
	}

	return svc
}

func (s *StockQuantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockQuantResponse, error) {
	return s.stockQuantRepo.List(ctx, req)
}

func (s *StockQuantService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.CountStockQuantResponse, error) {
	count, err := s.stockQuantRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.CountStockQuantResponse{Count: uint64(count)}, nil
}

func (s *StockQuantService) Get(ctx context.Context, req *inventoryV1.GetStockQuantRequest) (*inventoryV1.StockQuant, error) {
	return s.stockQuantRepo.Get(ctx, req)
}

// GetOverview 库存经营总览（看板聚合：仓库数/SKU数/库存总量/执行记录数/低库存清单）。
func (s *StockQuantService) GetOverview(ctx context.Context, req *inventoryV1.GetStockQuantOverviewRequest) (*inventoryV1.StockQuantOverview, error) {
	warehouseCount, err := s.warehouseRepo.Count(ctx, nil)
	if err != nil {
		return nil, err
	}
	skuCount, err := s.stockQuantRepo.CountDistinctSku(ctx)
	if err != nil {
		return nil, err
	}
	totalQty, err := s.stockQuantRepo.SumQuantity(ctx)
	if err != nil {
		return nil, err
	}
	movementCount, err := s.stockMoveLineRepo.CountAll(ctx)
	if err != nil {
		return nil, err
	}

	threshold := req.GetLowStockThreshold()
	if threshold <= 0 {
		threshold = defaultLowStockThreshold
	}
	limit := int(req.GetLowStockLimit())
	if limit <= 0 {
		limit = 10
	}
	lowStock, err := s.stockQuantRepo.ListLowStock(ctx, threshold, limit)
	if err != nil {
		return nil, err
	}

	return &inventoryV1.StockQuantOverview{
		WarehouseCount:  uint64(warehouseCount),
		SkuCount:        uint64(skuCount),
		TotalQuantity:   totalQty,
		MovementCount:   uint64(movementCount),
		LowStockItems:   lowStock,
	}, nil
}

// GetMovementTrend 近 30 日每日执行记录条数（看板折线图）。
func (s *StockQuantService) GetMovementTrend(ctx context.Context, req *inventoryV1.GetMovementTrendRequest) (*inventoryV1.MovementTrendResponse, error) {
	points, err := s.stockMoveLineRepo.MovementTrend(ctx)
	if err != nil {
		return nil, err
	}
	return &inventoryV1.MovementTrendResponse{Points: points}, nil
}
