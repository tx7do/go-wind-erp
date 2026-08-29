package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// ApprovalFlowService 审批流模板（admin BFF facade，纯委派 core）。
type ApprovalFlowService struct {
	adminV1.ApprovalFlowServiceHTTPServer

	log *log.Helper

	approvalFlowServiceClient approvalV1.ApprovalFlowServiceClient
}

func NewApprovalFlowService(
	ctx *bootstrap.Context,
	approvalFlowServiceClient approvalV1.ApprovalFlowServiceClient,
) *ApprovalFlowService {
	l := log.NewHelper(log.With(ctx.GetLogger(), "module", "approval_flow/service/admin-service"))
	return &ApprovalFlowService{
		log:                       l,
		approvalFlowServiceClient: approvalFlowServiceClient,
	}
}

func (s *ApprovalFlowService) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalFlowResponse, error) {
	return s.approvalFlowServiceClient.List(ctx, req)
}

func (s *ApprovalFlowService) Get(ctx context.Context, req *approvalV1.GetApprovalFlowRequest) (*approvalV1.ApprovalFlow, error) {
	return s.approvalFlowServiceClient.Get(ctx, req)
}

func (s *ApprovalFlowService) Create(ctx context.Context, req *approvalV1.CreateApprovalFlowRequest) (*emptypb.Empty, error) {
	return s.approvalFlowServiceClient.Create(ctx, req)
}

func (s *ApprovalFlowService) Update(ctx context.Context, req *approvalV1.UpdateApprovalFlowRequest) (*emptypb.Empty, error) {
	return s.approvalFlowServiceClient.Update(ctx, req)
}

func (s *ApprovalFlowService) Delete(ctx context.Context, req *approvalV1.DeleteApprovalFlowRequest) (*emptypb.Empty, error) {
	return s.approvalFlowServiceClient.Delete(ctx, req)
}
