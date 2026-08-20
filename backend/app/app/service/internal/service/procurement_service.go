package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
)

// PurchaseOrderService 采购单（app BFF facade，移动端只读：
// 审批人在审批中心看到采购审批后可查单据原文）。
type PurchaseOrderService struct {
	appV1.PurchaseOrderServiceHTTPServer

	purchaseOrderServiceClient procurementV1.PurchaseOrderServiceClient

	log *log.Helper
}

func NewPurchaseOrderService(
	ctx *bootstrap.Context,
	purchaseOrderServiceClient procurementV1.PurchaseOrderServiceClient,
) *PurchaseOrderService {
	return &PurchaseOrderService{
		log:                        ctx.NewLoggerHelper("purchase_order/service/app-service"),
		purchaseOrderServiceClient: purchaseOrderServiceClient,
	}
}

func (s *PurchaseOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListPurchaseOrderResponse, error) {
	return s.purchaseOrderServiceClient.List(ctx, req)
}

func (s *PurchaseOrderService) Get(ctx context.Context, req *procurementV1.GetPurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	return s.purchaseOrderServiceClient.Get(ctx, req)
}
