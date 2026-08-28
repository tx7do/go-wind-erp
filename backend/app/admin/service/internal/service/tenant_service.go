package service

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-kratos/kratos/v2/log"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/go-utils/captcha"

	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	authenticationV1 "go-wind-erp/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-erp/api/gen/go/identity/service/v1"
	permissionV1 "go-wind-erp/api/gen/go/permission/service/v1"

	"go-wind-erp/pkg/middleware/auth"
	"go-wind-erp/pkg/netutil"
)

type TenantService struct {
	adminV1.TenantServiceHTTPServer

	log *log.Helper

	userServiceClient           identityV1.UserServiceClient
	userCredentialServiceClient authenticationV1.UserCredentialServiceClient
	tenantServiceClient         identityV1.TenantServiceClient
	roleServiceClient           permissionV1.RoleServiceClient
	captchaClient               *captcha.Captcha
}

func NewTenantService(
	ctx *bootstrap.Context,
	userServiceClient identityV1.UserServiceClient,
	userCredentialServiceClient authenticationV1.UserCredentialServiceClient,
	tenantServiceClient identityV1.TenantServiceClient,
	roleServiceClient permissionV1.RoleServiceClient,
	captchaClient *captcha.Captcha,
) *TenantService {
	svc := &TenantService{
		log:                         ctx.NewLoggerHelper("tenant/service/admin-service"),
		userServiceClient:           userServiceClient,
		userCredentialServiceClient: userCredentialServiceClient,
		tenantServiceClient:         tenantServiceClient,
		roleServiceClient:           roleServiceClient,
		captchaClient:               captchaClient,
	}

	svc.init()

	return svc
}

func (s *TenantService) init() {
}

func (s *TenantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	resp, err := s.tenantServiceClient.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TenantService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountTenantResponse, error) {
	return s.tenantServiceClient.Count(ctx, req)
}

func (s *TenantService) Get(ctx context.Context, req *identityV1.GetTenantRequest) (*identityV1.Tenant, error) {
	resp, err := s.tenantServiceClient.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *TenantService) Create(ctx context.Context, req *identityV1.CreateTenantRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.CreatedBy = trans.Ptr(operator.UserId)

	if _, err = s.tenantServiceClient.Create(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) Update(ctx context.Context, req *identityV1.UpdateTenantRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Data.Id = trans.Ptr(req.GetId())

	req.Data.UpdatedBy = trans.Ptr(operator.GetUserId())
	if req.UpdateMask != nil {
		req.UpdateMask.Paths = append(req.UpdateMask.Paths, "updated_by")
	}

	return s.tenantServiceClient.Update(ctx, req)
}

func (s *TenantService) Delete(ctx context.Context, req *identityV1.DeleteTenantRequest) (*emptypb.Empty, error) {
	return s.tenantServiceClient.Delete(ctx, req)
}

func (s *TenantService) TenantExists(ctx context.Context, req *identityV1.TenantExistsRequest) (*identityV1.TenantExistsResponse, error) {
	return s.tenantServiceClient.TenantExists(ctx, req)
}

func (s *TenantService) CreateTenantWithAdminUser(ctx context.Context, req *identityV1.CreateTenantWithAdminUserRequest) (*emptypb.Empty, error) {
	if req == nil || req.Tenant == nil || req.User == nil {
		s.log.Error("invalid parameter: tenant or user is nil", req)
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	// 获取操作人信息
	operator, err := auth.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.Tenant.CreatedBy = trans.Ptr(operator.UserId)
	req.User.CreatedBy = trans.Ptr(operator.UserId)

	req.OperatorUserId = trans.Ptr(operator.GetUserId())

	return s.tenantServiceClient.CreateTenantWithAdminUser(ctx, req)
}

// SelfRegisterTenant 租户自助注册（公开端点）。
// 公开接口必须带验证码（请求头 X-Captcha-Id/Value，与登录同机制）防滥用；
// 校验通过后委派 core 完成租户+管理员+角色+业务引导数据的创建。
// 验证码开关与登录共用 CaptchaEnabled（开发环境可关闭）。
func (s *TenantService) SelfRegisterTenant(ctx context.Context, req *identityV1.SelfRegisterTenantRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, adminV1.ErrorBadRequest("invalid parameter")
	}

	if CaptchaEnabled && s.captchaClient != nil {
		header := netutil.HeaderFromContext(ctx)
		if header == nil {
			return nil, authenticationV1.ErrorBadRequest("invalid or missing captcha")
		}
		captchaID := strings.TrimSpace(header.Get(headerCaptchaID))
		captchaValue := strings.TrimSpace(header.Get(headerCaptchaValue))
		if captchaID == "" || captchaValue == "" {
			return nil, authenticationV1.ErrorBadRequest("invalid or missing captcha")
		}
		ok, err := s.captchaClient.Verify(ctx, captchaID, captchaValue)
		if err != nil || !ok {
			return nil, authenticationV1.ErrorBadRequest("invalid or missing captcha")
		}
	}

	return s.tenantServiceClient.SelfRegisterTenant(ctx, req)
}
