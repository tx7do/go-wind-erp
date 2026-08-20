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

// StockMovementService 库存流水服务
type StockMovementService struct {
	inventoryV1.UnimplementedStockMovementServiceServer

	log *log.Helper

	stockMovementRepo *data.StockMovementRepo
}

func NewStockMovementService(
	ctx *bootstrap.Context,
	stockMovementRepo *data.StockMovementRepo,
) *StockMovementService {
	svc := &StockMovementService{
		log:      ctx.NewLoggerHelper("stock_movement/service/core-service"),
		stockMovementRepo: stockMovementRepo,
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

	// 金额溢出校验：delta 与 quantity_before/quantity_after 的加法关系必须自洽，
	// 且 quantity_before + delta 必须等于 quantity_after，不得溢出 int64。
	qb := req.Data.GetQuantityBefore()
	d := req.Data.GetDelta()
	qa := req.Data.GetQuantityAfter()

	// 零变更是无意义流水，拒绝（避免污染台账）。
	if d == 0 {
		return nil, inventoryV1.ErrorBadRequest("delta must not be zero")
	}

	sum, overflow := addChecked(qb, d)
	if overflow {
		return nil, inventoryV1.ErrorBadRequest("quantity arithmetic overflow")
	}
	if sum != qa {
		return nil, inventoryV1.ErrorBadRequest("quantity_before + delta != quantity_after")
	}

	if _, err := s.stockMovementRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	// SAGA 桩：持久化成功后才发出库类补偿通知（避免"建单失败却发了事件"）；
	// 通知失败仅记录——当前为 no-op 桩，接真实实现时需评估补偿策略。
	if d < 0 {
		if err := defaultSagaSeam.notifyProcurement(ctx, stockEvent{
			warehouseCode: req.Data.GetWarehouseCode(),
			skuCode:       req.Data.GetSkuCode(),
			delta:         d,
		}); err != nil {
			s.log.Errorf("saga notify procurement failed: %s", err.Error())
		}
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
