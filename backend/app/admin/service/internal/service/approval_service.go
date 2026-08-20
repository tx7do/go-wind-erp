package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"

	"go-wind-erp/pkg/middleware/auth"
)

type ApprovalRequestService struct {
	adminV1.ApprovalRequestServiceHTTPServer

	log *log.Helper

	approvalRequestServiceClient approvalV1.ApprovalRequestServiceClient
}

func NewApprovalRequestService(
	ctx *bootstrap.Context,
	approvalRequestServiceClient approvalV1.ApprovalRequestServiceClient,
) *ApprovalRequestService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "approval_request/service/admin-service"))
	return &ApprovalRequestService{
		log:                          l,
		approvalRequestServiceClient: approvalRequestServiceClient,
	}
}

func (s *ApprovalRequestService) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalRequestResponse, error) {
	return s.approvalRequestServiceClient.List(ctx, req)
}

func (s *ApprovalRequestService) Get(ctx context.Context, req *approvalV1.GetApprovalRequestRequest) (*approvalV1.ApprovalRequest, error) {
	return s.approvalRequestServiceClient.Get(ctx, req)
}

func (s *ApprovalRequestService) Create(ctx context.Context, req *approvalV1.CreateApprovalRequestRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	_, err = s.approvalRequestServiceClient.Create(ctx, req)
	return &emptypb.Empty{}, err
}

func (s *ApprovalRequestService) Delete(ctx context.Context, req *approvalV1.DeleteApprovalRequestRequest) (*emptypb.Empty, error) {
	return s.approvalRequestServiceClient.Delete(ctx, req)
}

// Approve/Reject/Cancel：审批人与申请人校验均在 core 由 viewer context 推导，
// facade 纯委派。
func (s *ApprovalRequestService) Approve(ctx context.Context, req *approvalV1.ApproveApprovalRequestRequest) (*emptypb.Empty, error) {
	return s.approvalRequestServiceClient.Approve(ctx, req)
}

func (s *ApprovalRequestService) Reject(ctx context.Context, req *approvalV1.RejectApprovalRequestRequest) (*emptypb.Empty, error) {
	return s.approvalRequestServiceClient.Reject(ctx, req)
}

func (s *ApprovalRequestService) Cancel(ctx context.Context, req *approvalV1.CancelApprovalRequestRequest) (*emptypb.Empty, error) {
	return s.approvalRequestServiceClient.Cancel(ctx, req)
}
