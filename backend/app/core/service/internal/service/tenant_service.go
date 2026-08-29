package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	"github.com/tx7do/go-utils/aggregator"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-erp/app/core/service/internal/data"
	appViewer "go-wind-erp/pkg/entgo/viewer"

	authenticationV1 "go-wind-erp/api/gen/go/authentication/service/v1"
	identityV1 "go-wind-erp/api/gen/go/identity/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
	productV1 "go-wind-erp/api/gen/go/product/service/v1"
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"

	"go-wind-erp/pkg/constants"
	permissionV1 "go-wind-erp/api/gen/go/permission/service/v1"
)

type TenantService struct {
	identityV1.UnimplementedTenantServiceServer

	log *log.Helper

	tenantRepo          *data.TenantRepo
	userRepo            data.UserRepo
	userCredentialsRepo *data.UserCredentialRepo
	roleRepo            *data.RoleRepo
	locationRepo        *data.LocationRepo
	warehouseService    *WarehouseService
	productService      *ProductService
	supplierService     *SupplierService
	customerService     *CustomerService
}

func NewTenantService(
	ctx *bootstrap.Context,
	tenantRepo *data.TenantRepo,
	userRepo data.UserRepo,
	userCredentialsRepo *data.UserCredentialRepo,
	roleRepo *data.RoleRepo,
	locationRepo *data.LocationRepo,
	warehouseService *WarehouseService,
	productService *ProductService,
	supplierService *SupplierService,
	customerService *CustomerService,
) *TenantService {
	return &TenantService{
		log:                 ctx.NewLoggerHelper("tenant/service/core-service"),
		tenantRepo:          tenantRepo,
		userRepo:            userRepo,
		userCredentialsRepo: userCredentialsRepo,
		roleRepo:            roleRepo,
		locationRepo:        locationRepo,
		warehouseService:    warehouseService,
		productService:      productService,
		supplierService:     supplierService,
		customerService:     customerService,
	}
}

func (s *TenantService) extractRelationIDs(
	tenants []*identityV1.Tenant,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	for _, t := range tenants {
		if t.GetAdminUserId() > 0 {
			userSet[t.GetAdminUserId()] = nil
		}
	}
}

func (s *TenantService) fetchRelationInfo(
	ctx context.Context,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) error {
	if len(userSet) > 0 {
		userIds := make([]uint32, 0, len(userSet))
		for id := range userSet {
			userIds = append(userIds, id)
		}

		users, err := s.userRepo.ListUsersByIds(ctx, userIds)
		if err != nil {
			s.log.Errorf("query users err: %v", err)
			return err
		}

		for _, u := range users {
			userSet[u.GetId()] = u
		}
	}

	return nil
}

func (s *TenantService) bindRelations(
	tenants []*identityV1.Tenant,
	userSet aggregator.ResourceMap[uint32, *identityV1.User],
) {
	aggregator.Populate(
		tenants,
		userSet,
		func(ou *identityV1.Tenant) uint32 { return ou.GetAdminUserId() },
		func(ou *identityV1.Tenant, r *identityV1.User) {
			ou.AdminUserName = r.Username
		},
	)
}

func (s *TenantService) enrichRelations(ctx context.Context, tenants []*identityV1.Tenant) error {
	var userSet = make(aggregator.ResourceMap[uint32, *identityV1.User])
	s.extractRelationIDs(tenants, userSet)
	if err := s.fetchRelationInfo(ctx, userSet); err != nil {
		return err
	}
	s.bindRelations(tenants, userSet)
	return nil
}

func (s *TenantService) List(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.ListTenantResponse, error) {
	resp, err := s.tenantRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.enrichRelations(ctx, resp.Items)

	return resp, nil
}

func (s *TenantService) Count(ctx context.Context, req *paginationV1.PagingRequest) (*identityV1.CountTenantResponse, error) {
	count, err := s.tenantRepo.Count(ctx, req)
	if err != nil {
		return nil, err
	}

	return &identityV1.CountTenantResponse{
		Count: uint64(count),
	}, nil
}

func (s *TenantService) Get(ctx context.Context, req *identityV1.GetTenantRequest) (*identityV1.Tenant, error) {
	resp, err := s.tenantRepo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	fakeItems := []*identityV1.Tenant{resp}
	_ = s.enrichRelations(ctx, fakeItems)

	return resp, nil
}

// ResolveTenantByDomain 按域名解析租户ID。
//
// 供 app BFF 在匿名（白名单）请求中按 Host 解析 tenant_id。仅返回 id，不映射
// 其他租户字段；未匹配到时返回 tenant_id=0，调用方据此 fail-closed（拒绝而非
// 回退 SystemViewer，避免跨租户泄漏）。
func (s *TenantService) ResolveTenantByDomain(ctx context.Context, req *identityV1.ResolveTenantByDomainRequest) (*identityV1.ResolveTenantByDomainResponse, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}
	tid, err := s.tenantRepo.GetTenantIdByDomain(ctx, req.GetDomain())
	if err != nil {
		return nil, err
	}
	return &identityV1.ResolveTenantByDomainResponse{TenantId: tid}, nil
}

func (s *TenantService) Create(ctx context.Context, req *identityV1.CreateTenantRequest) (*identityV1.Tenant, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	var tenant *identityV1.Tenant
	var err error
	if tenant, err = s.tenantRepo.Create(ctx, req.Data); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *TenantService) Update(ctx context.Context, req *identityV1.UpdateTenantRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	if err := s.tenantRepo.Update(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) Delete(ctx context.Context, req *identityV1.DeleteTenantRequest) (*emptypb.Empty, error) {
	if err := s.tenantRepo.Delete(ctx, req); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (s *TenantService) TenantExists(ctx context.Context, req *identityV1.TenantExistsRequest) (*identityV1.TenantExistsResponse, error) {
	return s.tenantRepo.TenantExists(ctx, req)
}

// CreateTenantWithAdminUser 创建租户及其管理员用户
func (s *TenantService) CreateTenantWithAdminUser(ctx context.Context, req *identityV1.CreateTenantWithAdminUserRequest) (ret *emptypb.Empty, err error) {
	if req == nil || req.Tenant == nil || req.User == nil {
		s.log.Error("invalid parameter: tenant or user is nil", req)
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	// Check if tenant code or admin username already exists
	if _, err = s.tenantRepo.TenantExists(ctx, &identityV1.TenantExistsRequest{
		Code: req.GetTenant().GetCode(),
		Name: req.GetTenant().GetName(),
	}); err != nil {
		s.log.Errorf("check tenant code exists err: %v", err)
		return nil, err
	}

	// 注意：此处不再做 admin username 的全局查重。
	// username 仅在 (tenant_id, username) 维度唯一（见 ent/schema/user.go 唯一索引），
	// 不同租户允许使用相同 username。此前按 username 跨租户全局查重会误判冲突、
	// 且在平台上下文（tid=0)下跨租户泄露用户名存在性。租户内唯一性由 DB 唯一索引保证。

	tx, err := s.tenantRepo.BeginTx(ctx)
	if err != nil {
		s.log.Errorf("begin tx err: %v", err)
		return nil, err
	}
	defer func() { s.tenantRepo.FinishTx(tx, err) }()

	// CreateTranslation tenant
	var tenant *identityV1.Tenant
	if tenant, err = s.tenantRepo.CreateWithTx(ctx, tx, req.Tenant); err != nil {
		s.log.Errorf("create tenant err: %v", err)
		return nil, err
	}

	req.User.TenantId = tenant.Id

	// copy tenant manager role to tenant
	// 操作人身份必须从 viewer context 推导，忽略客户端传入的 operator_user_id，
	// 防止伪造审计归属
	var operatorUserID uint32
	if opID, hasOp := viewerUserIDFromContext(ctx); hasOp {
		operatorUserID = opID
	}
	var role *permissionV1.Role
	if role, err = s.roleRepo.CreateTenantRoleFromTemplate(ctx, tx, tenant.GetId(), operatorUserID); err != nil {
		s.log.Errorf("copy tenant admin role template to tenant err: %v", err)
		return nil, err
	}

	// CreateTranslation tenant admin user
	var adminUser *identityV1.User
	req.User.RoleId = role.Id
	//req.User.Status = identityV1.User_NORMAL.Enum()
	if adminUser, err = s.userRepo.CreateWithTx(ctx, tx, req.User); err != nil {
		s.log.Errorf("create tenant admin user err: %v", err)
		return nil, err
	}

	// CreateTranslation user credential
	if err = s.userCredentialsRepo.CreateWithTx(ctx, tx, &authenticationV1.UserCredential{
		UserId:         adminUser.Id,
		TenantId:       tenant.Id,
		IdentityType:   authenticationV1.UserCredential_USERNAME.Enum(),
		Identifier:     adminUser.Username,
		CredentialType: authenticationV1.UserCredential_PASSWORD_HASH.Enum(),
		Credential:     trans.Ptr(req.GetPassword()),
		IsPrimary:      trans.Ptr(true),
		Status:         authenticationV1.UserCredential_ENABLED.Enum(),
	}); err != nil {
		s.log.Errorf("create tenant admin user credential err: %v", err)
		return nil, err
	}

	// assign admin user id to tenant
	if err = s.tenantRepo.AssignTenantAdmin(ctx, tx, *tenant.Id, *adminUser.Id); err != nil {
		s.log.Errorf("assign admin user id to tenant err: %v", err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// SelfRegisterTenant 租户自助注册（公开端点）：
// 复用 CreateTenantWithAdminUser 的完整事务（租户+角色克隆+管理员+凭证+绑定），
// 注册成功后追加业务引导：SUPPLIER/CUSTOMER 虚拟位置 + 默认仓库（自动带 INTERNAL
// 接收位置），使新租户注册即可创建采购单/销售单（半天自助上线的后端保障）。
// 引导失败不回滚注册（租户与管理员已是事实），仅记录日志可重试。
func (s *TenantService) SelfRegisterTenant(ctx context.Context, req *identityV1.SelfRegisterTenantRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, identityV1.ErrorBadRequest("invalid parameter")
	}

	tenantName := strings.TrimSpace(req.GetTenantName())
	tenantCode := strings.TrimSpace(req.GetTenantCode())
	adminUsername := strings.TrimSpace(req.GetAdminUsername())
	password := req.GetPassword()

	if tenantName == "" || tenantCode == "" {
		return nil, identityV1.ErrorBadRequest("tenant_name and tenant_code are required")
	}
	if adminUsername == "" {
		return nil, identityV1.ErrorBadRequest("admin_username is required")
	}
	if len(password) < 6 {
		return nil, identityV1.ErrorBadRequest("password must be at least 6 characters")
	}

	// 自助注册即时可用：ON + APPROVED + PAID（对齐演示租户的形态）。
	onStatus := identityV1.Tenant_ON
	paidType := identityV1.Tenant_PAID
	approved := identityV1.Tenant_APPROVED

	userStatus := identityV1.User_NORMAL

	if _, err := s.CreateTenantWithAdminUser(ctx, &identityV1.CreateTenantWithAdminUserRequest{
		Tenant: &identityV1.Tenant{
			Name:        trans.Ptr(tenantName),
			Code:        trans.Ptr(tenantCode),
			Type:        &paidType,
			Status:      &onStatus,
			AuditStatus: &approved,
			Remark:      trans.Ptr("self-registered tenant"),
		},
		User: &identityV1.User{
			Username: trans.Ptr(adminUsername),
			Nickname: trans.Ptr(adminUsername),
			Status:   &userStatus,
		},
		Password: password,
	}); err != nil {
		return nil, err
	}

	// 业务引导：以新租户的匿名 viewer 上下文创建虚拟位置与默认仓库。
	tid, err := s.resolveTenantIDByCode(ctx, tenantCode)
	if err != nil {
		// 注册已成功，引导数据缺失仅记录（管理端可补救），不阻断返回。
		s.log.Errorf("self register: resolve tenant id by code failed: %v", err)
	} else {
		tenantCtx := appViewer.NewAnonymousTenantViewerContext(ctx, tid, "self-register")

		if berr := s.locationRepo.CreateSupplierLocation(tenantCtx); berr != nil {
			s.log.Errorf("self register: create supplier location failed: %v", berr)
		}
		if berr := s.locationRepo.CreateCustomerLocation(tenantCtx); berr != nil {
			s.log.Errorf("self register: create customer location failed: %v", berr)
		}
		if berr := s.locationRepo.CreateInventoryLossLocation(tenantCtx); berr != nil {
			s.log.Errorf("self register: create inventory loss location failed: %v", berr)
		}
		if _, berr := s.warehouseService.Create(tenantCtx, &inventoryV1.CreateWarehouseRequest{
			Data: &inventoryV1.Warehouse{
				Code:   trans.Ptr("MAIN"),
				Name:   trans.Ptr("默认仓库"),
				Enable: trans.Ptr(true),
				Remark: trans.Ptr("self-registered default warehouse"),
			},
		}); berr != nil {
			s.log.Errorf("self register: create default warehouse failed: %v", berr)
		}

		// 演示数据（半天上线的"开箱即用"）：商品/供应商/客户各一小套，
		// 注册后立即可开单体验；均为普通数据，可直接改名改码或删除。
		s.seedDemoData(tenantCtx)
	}

	s.log.Infof("self register: tenant %q registered with admin %q", tenantCode, adminUsername)

	return &emptypb.Empty{}, nil
}

// resolveTenantIDByCode 按编码查租户 ID（注册后引导数据需要租户上下文）。
func (s *TenantService) resolveTenantIDByCode(ctx context.Context, code string) (uint32, error) {
	resp, err := s.tenantRepo.List(ctx, &paginationV1.PagingRequest{
		NoPaging: trans.Ptr(true),
		FilteringType: &paginationV1.PagingRequest_Query{
			Query: fmt.Sprintf(`{"code": "%s"}`, code),
		},
	})
	if err != nil {
		return 0, err
	}
	if len(resp.GetItems()) == 0 {
		return 0, fmt.Errorf("tenant %q not found after registration", code)
	}
	return resp.GetItems()[0].GetId(), nil
}


// seedDemoData 演示数据引导（全部软失败：缺失仅记录，不阻断注册）。
func (s *TenantService) seedDemoData(tenantCtx context.Context) {
	for _, p := range constants.DemoProducts {
		if _, err := s.productService.Create(tenantCtx, &productV1.CreateProductRequest{
			Data: &productV1.Product{
				Code:   trans.Ptr(p.Code),
				Name:   trans.Ptr(p.Name),
				Spec:   trans.Ptr(p.Spec),
				Unit:   trans.Ptr(p.Unit),
				Enable: trans.Ptr(true),
			},
		}); err != nil {
			s.log.Errorf("self register: seed demo product %s failed: %v", p.Code, err)
		}
	}
	for _, sp := range constants.DemoSuppliers {
		if _, err := s.supplierService.Create(tenantCtx, &procurementV1.CreateSupplierRequest{
			Data: &procurementV1.Supplier{
				Code:    trans.Ptr(sp.Code),
				Name:    trans.Ptr(sp.Name),
				Contact: trans.Ptr(sp.Contact),
				Phone:   trans.Ptr(sp.Phone),
				Enable:  trans.Ptr(true),
			},
		}); err != nil {
			s.log.Errorf("self register: seed demo supplier %s failed: %v", sp.Code, err)
		}
	}
	for _, c := range constants.DemoCustomers {
		if _, err := s.customerService.Create(tenantCtx, &salesV1.CreateCustomerRequest{
			Data: &salesV1.Customer{
				Code:    trans.Ptr(c.Code),
				Name:    trans.Ptr(c.Name),
				Contact: trans.Ptr(c.Contact),
				Phone:   trans.Ptr(c.Phone),
				Enable:  trans.Ptr(true),
			},
		}); err != nil {
			s.log.Errorf("self register: seed demo customer %s failed: %v", c.Code, err)
		}
	}
}
