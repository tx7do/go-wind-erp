package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	appV1 "go-wind-erp/api/gen/go/app/service/v1"
	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

type ApprovalRequestService struct {
	appV1.ApprovalRequestServiceHTTPServer

	approvalRequestServiceClient approvalV1.ApprovalRequestServiceClient

	log *log.Helper
}

func NewApprovalRequestService(
	ctx *bootstrap.Context,
	approvalRequestServiceClient approvalV1.ApprovalRequestServiceClient,
) *ApprovalRequestService {
	return &ApprovalRequestService{
		log:                          ctx.NewLoggerHelper("approval_request/service/app-service"),
		approvalRequestServiceClient: approvalRequestServiceClient,
	}
}

func (s *ApprovalRequestService) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalRequestResponse, error) {
	return s.approvalRequestServiceClient.List(ctx, req)
}

func (s *ApprovalRequestService) Get(ctx context.Context, req *approvalV1.GetApprovalRequestRequest) (*approvalV1.ApprovalRequest, error) {
	return s.approvalRequestServiceClient.Get(ctx, req)
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
