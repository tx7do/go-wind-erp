package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-erp/app/core/service/internal/data"

	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
)

// StockLotService 批次服务（只读）：批次余量与效期状态由仓储从 move lines
// 聚合推导（记录式批次/效期，批次仅在收货 Validate 时经 get-or-create 登记）。
type StockLotService struct {
	inventoryV1.UnimplementedStockLotServiceServer

	log          *log.Helper
	stockLotRepo *data.StockLotRepo
}

func NewStockLotService(
	ctx *bootstrap.Context,
	stockLotRepo *data.StockLotRepo,
) *StockLotService {
	return &StockLotService{
		log:          ctx.NewLoggerHelper("stock_lot/service/core-service"),
		stockLotRepo: stockLotRepo,
	}
}

// List 批次库存列表（含剩余数量与效期状态；skuCode/lotStatus 过滤）。
func (s *StockLotService) List(ctx context.Context, req *paginationV1.PagingRequest) (*inventoryV1.ListStockLotResponse, error) {
	return s.stockLotRepo.List(ctx, req)
}
