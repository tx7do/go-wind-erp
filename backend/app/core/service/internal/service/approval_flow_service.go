package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
)

// ApprovalFlowService 审批流模板服务（多级审批配置）。
//
// 每租户每业务类型至多一条生效流程（服务层唯一性校验）；级定义随流程
// 整体替换。提交业务单时由 ApprovalRequestService.Create 快照级数进
// 请求，在途审批不受流程编辑影响。
type ApprovalFlowService struct {
	approvalV1.UnimplementedApprovalFlowServiceServer

	log             *log.Helper
	approvalFlowRepo *data.ApprovalFlowRepo
}

func NewApprovalFlowService(
	ctx *bootstrap.Context,
	approvalFlowRepo *data.ApprovalFlowRepo,
) *ApprovalFlowService {
	return &ApprovalFlowService{
		log:              ctx.NewLoggerHelper("approval_flow/service/core-service"),
		approvalFlowRepo: approvalFlowRepo,
	}
}

func (s *ApprovalFlowService) List(ctx context.Context, req *paginationV1.PagingRequest) (*approvalV1.ListApprovalFlowResponse, error) {
	return s.approvalFlowRepo.List(ctx, req)
}

func (s *ApprovalFlowService) Get(ctx context.Context, req *approvalV1.GetApprovalFlowRequest) (*approvalV1.ApprovalFlow, error) {
	if req == nil || req.GetId() == 0 {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}
	return s.approvalFlowRepo.Get(ctx, req.GetId())
}

var validFlowBizTypes = map[string]bool{
	"PURCHASE_ORDER": true,
	"SALES_ORDER":    true,
	"PAYMENT":        true,
	"RECEIPT":        true,
}

func (s *ApprovalFlowService) Create(ctx context.Context, req *approvalV1.CreateApprovalFlowRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}
	if req.Data.GetBizType() == "" || !validFlowBizTypes[req.Data.GetBizType()] {
		return nil, approvalV1.ErrorBadRequest("unsupported biz_type: %s", req.Data.GetBizType())
	}
	if len(req.Data.GetSteps()) == 0 {
		return nil, approvalV1.ErrorBadRequest("approval flow requires at least one step")
	}

	// 同租户同业务类型唯一
	exists, err := s.approvalFlowRepo.ExistsActiveByBizType(ctx, req.Data.GetBizType(), 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, approvalV1.ErrorConflict(
			"业务类型 %s 已存在审批流，每类至多一条", req.Data.GetBizType())
	}

	if err := s.approvalFlowRepo.Create(ctx, req.Data); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *ApprovalFlowService) Update(ctx context.Context, req *approvalV1.UpdateApprovalFlowRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil || req.Data.GetId() == 0 {
		return nil, approvalV1.ErrorBadRequest("invalid parameter")
	}
	if len(req.Data.GetSteps()) == 0 {
		return nil, approvalV1.ErrorBadRequest("approval flow requires at least one step")
	}
	if req.Data.GetBizType() != "" && !validFlowBizTypes[req.Data.GetBizType()] {
		return nil, approvalV1.ErrorBadRequest("unsupported biz_type: %s", req.Data.GetBizType())
	}

	exists, err := s.approvalFlowRepo.ExistsActiveByBizType(ctx, req.Data.GetBizType(), req.Data.GetId())
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, approvalV1.ErrorConflict(
			"业务类型 %s 已存在其它审批流", req.Data.GetBizType())
	}

	if err := s.approvalFlowRepo.Update(ctx, req.Data); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *ApprovalFlowService) Delete(ctx context.Context, req *approvalV1.DeleteApprovalFlowRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, approvalV1.ErrorBadRequest("invalid request")
	}
	if err := s.approvalFlowRepo.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
