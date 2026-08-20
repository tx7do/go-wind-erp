package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

// SupplierService 供应商管理（admin BFF facade）
type SupplierService struct {
	adminV1.SupplierServiceHTTPServer

	log *log.Helper

	supplierServiceClient procurementV1.SupplierServiceClient
}

func NewSupplierService(
	ctx *bootstrap.Context,
	supplierServiceClient procurementV1.SupplierServiceClient,
) *SupplierService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "supplier/service/admin-service"))
	return &SupplierService{
		log:                   l,
		supplierServiceClient: supplierServiceClient,
	}
}

func (s *SupplierService) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListSupplierResponse, error) {
	return s.supplierServiceClient.List(ctx, req)
}

func (s *SupplierService) Get(ctx context.Context, req *procurementV1.GetSupplierRequest) (*procurementV1.Supplier, error) {
	return s.supplierServiceClient.Get(ctx, req)
}

func (s *SupplierService) Create(ctx context.Context, req *procurementV1.CreateSupplierRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.supplierServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *SupplierService) Update(ctx context.Context, req *procurementV1.UpdateSupplierRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	_, err = s.supplierServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *SupplierService) Delete(ctx context.Context, req *procurementV1.DeleteSupplierRequest) (*emptypb.Empty, error) {
	return s.supplierServiceClient.Delete(ctx, req)
}

// PurchaseOrderService 采购单管理（admin BFF facade）
type PurchaseOrderService struct {
	adminV1.PurchaseOrderServiceHTTPServer

	log *log.Helper

	purchaseOrderServiceClient procurementV1.PurchaseOrderServiceClient
}

func NewPurchaseOrderService(
	ctx *bootstrap.Context,
	purchaseOrderServiceClient procurementV1.PurchaseOrderServiceClient,
) *PurchaseOrderService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "purchase_order/service/admin-service"))
	return &PurchaseOrderService{
		log:                        l,
		purchaseOrderServiceClient: purchaseOrderServiceClient,
	}
}

func (s *PurchaseOrderService) List(ctx context.Context, req *paginationV1.PagingRequest) (*procurementV1.ListPurchaseOrderResponse, error) {
	return s.purchaseOrderServiceClient.List(ctx, req)
}

func (s *PurchaseOrderService) Get(ctx context.Context, req *procurementV1.GetPurchaseOrderRequest) (*procurementV1.PurchaseOrder, error) {
	return s.purchaseOrderServiceClient.Get(ctx, req)
}

func (s *PurchaseOrderService) Create(ctx context.Context, req *procurementV1.CreatePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)
	req.Data.Id = nil

	_, err = s.purchaseOrderServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *PurchaseOrderService) Update(ctx context.Context, req *procurementV1.UpdatePurchaseOrderRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())
	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	_, err = s.purchaseOrderServiceClient.Update(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *PurchaseOrderService) Delete(ctx context.Context, req *procurementV1.DeletePurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Delete(ctx, req)
}

// 动作类（submit/approve/reject/cancel/complete）：发起人与自审守卫均在
// core 由 viewer context 推导，facade 纯委派。
func (s *PurchaseOrderService) Submit(ctx context.Context, req *procurementV1.SubmitPurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Submit(ctx, req)
}

func (s *PurchaseOrderService) Approve(ctx context.Context, req *procurementV1.ApprovePurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Approve(ctx, req)
}

func (s *PurchaseOrderService) Reject(ctx context.Context, req *procurementV1.RejectPurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Reject(ctx, req)
}

func (s *PurchaseOrderService) Cancel(ctx context.Context, req *procurementV1.CancelPurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Cancel(ctx, req)
}

func (s *PurchaseOrderService) Complete(ctx context.Context, req *procurementV1.CompletePurchaseOrderRequest) (*emptypb.Empty, error) {
	return s.purchaseOrderServiceClient.Complete(ctx, req)
}
