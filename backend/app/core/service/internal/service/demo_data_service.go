package service

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"

	identityV1 "go-wind-erp/api/gen/go/identity/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	productV1 "go-wind-erp/api/gen/go/product/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"

	"go-wind-erp/app/core/service/internal/data"
	appViewer "go-wind-erp/pkg/entgo/viewer"
)

// DemoDataService 在开发/演示环境下灌入一份最小可用的业务演示数据：
// 一个演示租户 + 私户管理员（admin/admin），以及该租户下的若干仓库 / 产品 /
// 库存行 / 跨近 30 日的库存流水（填充看板折线图）。仅当环境变量
// ERP_DEMO_SEED=true 且演示租户尚不存在时执行，幂等。
//
// 禁用方式：不设或设 ERP_DEMO_SEED=任何非 true 值。生产环境务必不置为 true。
type DemoDataService struct {
	log *log.Helper

	tenantService    *TenantService
	warehouseService *WarehouseService
	productService   *ProductService
	supplierService  *SupplierService
	inventoryService *InventoryService
	stockMovementRepo *data.StockMovementRepo
}

func NewDemoDataService(
	ctx *bootstrap.Context,
	tenantService *TenantService,
	warehouseService *WarehouseService,
	productService *ProductService,
	supplierService *SupplierService,
	inventoryService *InventoryService,
	stockMovementRepo *data.StockMovementRepo,
) *DemoDataService {
	svc := &DemoDataService{
		log:               ctx.NewLoggerHelper("demo-data/service/core-service"),
		tenantService:     tenantService,
		warehouseService:  warehouseService,
		productService:    productService,
		supplierService:   supplierService,
		inventoryService:  inventoryService,
		stockMovementRepo: stockMovementRepo,
	}
	svc.init()
	return svc
}

func (s *DemoDataService) init() {
	if os.Getenv("ERP_DEMO_SEED") != "true" {
		return
	}
	ctx := appViewer.NewSystemViewerContext(context.Background())

	if exists, _ := s.tenantService.TenantExists(ctx, &identityV1.TenantExistsRequest{
		Code: demoTenantCode,
		Name: demoTenantName,
	}); exists.GetExist() {
		s.log.Infof("demo seed skipped: tenant %q already exists", demoTenantCode)
		return
	}

	s.log.Infof("demo seed: creating tenant %q + admin user", demoTenantCode)
	if err := s.seedTenantAndAdmin(ctx); err != nil {
		s.log.Errorf("demo seed tenant/admin failed: %v", err)
		return
	}
	if err := s.seedDomain(ctx); err != nil {
		s.log.Errorf("demo seed domain failed: %v", err)
	}
	s.log.Infof("demo seed complete")
}

func (s *DemoDataService) seedTenantAndAdmin(ctx context.Context) error {
	onStatus := identityV1.Tenant_ON
	paidType := identityV1.Tenant_PAID
	approved := identityV1.Tenant_APPROVED

	tenant := &identityV1.Tenant{
		Name:        strPtr(demoTenantName),
		Code:        strPtr(demoTenantCode),
		Type:        &paidType,
		Status:      &onStatus,
		AuditStatus: &approved,
		Industry:    strPtr("Retail"),
		Remark:      strPtr("demo seed tenant (auto-generated)"),
	}

	username := strPtr(demoAdminUsername)
	nickname := strPtr(demoAdminUsername)
	userStatus := identityV1.User_NORMAL
	adminUser := &identityV1.User{
		Username: username,
		Nickname: nickname,
		Email:    strPtr("demo-admin@example.invalid"),
		Status:   &userStatus,
	}

	if _, err := s.tenantService.CreateTenantWithAdminUser(ctx, &identityV1.CreateTenantWithAdminUserRequest{
		Tenant:   tenant,
		User:     adminUser,
		Password: demoAdminPassword,
	}); err != nil {
		return fmt.Errorf("create tenant with admin: %w", err)
	}
	return nil
}

func (s *DemoDataService) seedDomain(ctx context.Context) error {
	// 查到演示租户的 id 后切换为该租户作用域的匿名 viewer，使仓库 / 产品 /
	// 库存 / 流水按租户隔离写入。TenantService.List 默认走 SystemViewer 仍可
	// 跨租户列出。
	tenantID, err := s.resolveDemoTenantID(ctx)
	if err != nil {
		return err
	}
	tenantCtx := appViewer.NewAnonymousTenantViewerContext(context.Background(), tenantID, "demo-seed")

	for i := 0; i < demoWarehouseCount; i++ {
		code := fmt.Sprintf("WH-%02d", i+1)
		name := fmt.Sprintf("Demo Warehouse %d", i+1)
		enable := true
		if _, err := s.warehouseService.Create(tenantCtx, &inventoryV1.CreateWarehouseRequest{
			Data: &inventoryV1.Warehouse{
				TenantId: demoU32Ptr(tenantID),
				Code:     &code,
				Name:     &name,
				Enable:   &enable,
				Remark:   strPtr("demo seed"),
			},
		}); err != nil {
			return fmt.Errorf("seed warehouse %s: %w", code, err)
		}
	}

	for i := 0; i < demoProductCount; i++ {
		code := fmt.Sprintf("SKU-%04d", i+1)
		name := fmt.Sprintf("Demo Product %d", i+1)
		enable := true
		if _, err := s.productService.Create(tenantCtx, &productV1.CreateProductRequest{
			Data: &productV1.Product{
				TenantId: demoU32Ptr(tenantID),
				Code:     &code,
				Name:     &name,
				Enable:   &enable,
				Remark:   strPtr("demo seed"),
			},
		}); err != nil {
			return fmt.Errorf("seed product %s: %w", code, err)
		}
	}

	for i := 0; i < demoSupplierCount; i++ {
		code := fmt.Sprintf("SUP-%02d", i+1)
		name := fmt.Sprintf("Demo Supplier %d", i+1)
		enable := true
		if _, err := s.supplierService.Create(tenantCtx, &procurementV1.CreateSupplierRequest{
			Data: &procurementV1.Supplier{
				TenantId: demoU32Ptr(tenantID),
				Code:     &code,
				Name:     &name,
				Enable:   &enable,
				Remark:   strPtr("demo seed"),
			},
		}); err != nil {
			return fmt.Errorf("seed supplier %s: %w", code, err)
		}
	}

	// 库存行：每个仓库 × 第一个产品各一行，初始数量 100。
	status := inventoryV1.Inventory_AVAILABLE
	qty := int64(100)
	for w := 0; w < demoWarehouseCount; w++ {
		whCode := fmt.Sprintf("WH-%02d", w+1)
		sku := "SKU-0001"
		if _, err := s.inventoryService.Create(tenantCtx, &inventoryV1.CreateInventoryRequest{
			Data: &inventoryV1.Inventory{
				WarehouseCode: &whCode,
				SkuCode:       &sku,
				Quantity:      &qty,
				Status:        &status,
			},
		}); err != nil {
			return fmt.Errorf("seed inventory %s/%s: %w", whCode, sku, err)
		}
	}

	// 库存流水：近 30 日每仓库每产品各若干条，填充看板折线图。
	if err := s.stockMovementRepo.SeedDemoMovements(tenantCtx, tenantID); err != nil {
		return fmt.Errorf("seed movements: %w", err)
	}
	return nil
}

func (s *DemoDataService) resolveDemoTenantID(ctx context.Context) (uint32, error) {
	// 创建刚返回 emptypb.Empty，拿不到 id；以 NoPaging 列出租户后按 code 匹配
	// 取回新建演示租户的 id，用于切换租户作用域 viewer。
	noPaging := true
	list, err := s.tenantService.List(ctx, &paginationV1.PagingRequest{NoPaging: &noPaging})
	if err != nil {
		return 0, fmt.Errorf("list tenants: %w", err)
	}
	for _, t := range list.GetItems() {
		if t.GetCode() == demoTenantCode {
			return t.GetId(), nil
		}
	}
	return 0, fmt.Errorf("demo tenant %q not found after creation", demoTenantCode)
}

func demoU32Ptr(v uint32) *uint32 { return &v }

const (
	demoTenantCode     = "demo"
	demoTenantName     = "演示租户"
	demoAdminUsername  = "admin"
	demoAdminPassword  = "admin"
	demoWarehouseCount = 3
	demoProductCount   = 5
	demoSupplierCount  = 2
)

