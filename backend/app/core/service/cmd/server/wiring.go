package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"go-wind-erp/app/core/service/internal/data"
	"go-wind-erp/app/core/service/internal/data/client"
	"go-wind-erp/app/core/service/internal/server"
	"go-wind-erp/app/core/service/internal/service"
)

// initApp 手写装配整个应用,是本服务的依赖注入点(替代原 google/wire 生成)。
// initApp assembles the whole application by hand — the dependency injection point.
//
// 装配严格单向分层,自上而下的阅读顺序即依赖方向:
// The wiring is strictly layered; reading top-down follows the dependency direction:
//
//	基础设施 → 仓储层(data) → 服务层(service) → 传输层(server)
//
// 约定 / Conventions:
//   - 新增仓储/服务:在对应分层小节追加一行构造,并传给下游消费者;漏接由编译器在调用处报错。
//     To add a repo/service: append one constructor line in its layer section, then pass it to
//     downstream consumers; a missing connection is a compile error at the call site.
//   - 持有 cleanup 的资源创建成功后立即注册进 cleanups;任何一步失败,rollback 逆序执行已注册
//     的清理(shutdown 与中途失败共用同一条 LIFO 路径)。
//     Resources owning a cleanup register it immediately; rollback runs them LIFO both on
//     mid-way failure and on shutdown.
//   - 本文件只做构造与传参,不写业务逻辑。
//     Construction and parameter passing only; no business logic in this file.
func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	// cleanup 注册表:rollback 时逆序执行。
	// Cleanup registry; rollback runs entries in reverse order.
	var cleanups []func()
	rollback := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// ═══════════════════════ 一、基础设施 ═══════════════════════

	authenticatorOption := data.NewAuthenticatorConfig(ctx)
	redisClient, cleanupRedis, err := client.NewRedisClient(ctx)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	cleanups = append(cleanups, cleanupRedis)

	userTokenCache := data.NewUserTokenCache(ctx, redisClient)
	authenticator := data.NewAuthenticator(ctx, authenticatorOption, userTokenCache)
	entClient, cleanupEnt, err := client.NewEntClient(ctx)
	if err != nil {
		rollback()
		return nil, nil, err
	}
	cleanups = append(cleanups, cleanupEnt)

	crypto := data.NewPasswordCrypto()
	minIOClient := client.NewMinIoClient(ctx)
	contactCodeCache := data.NewContactCodeCache(ctx, redisClient)

	// ═══════════════════════ 二、仓储层(internal/data) ═══════════════════════

	userCredentialRepo := data.NewUserCredentialRepo(ctx, entClient, crypto)
	userRoleRepo := data.NewUserRoleRepo(ctx, entClient)
	userOrgUnitRepo := data.NewUserOrgUnitRepo(ctx, entClient)
	userPositionRepo := data.NewUserPositionRepo(ctx, entClient)
	membershipRoleRepo := data.NewMembershipRoleRepo(ctx, entClient)
	membershipPositionRepo := data.NewMembershipPositionRepo(ctx, entClient)
	membershipOrgUnitRepo := data.NewMembershipOrgUnitRepo(ctx, entClient)
	membershipRepo := data.NewMembershipRepo(ctx, entClient, membershipRoleRepo, membershipPositionRepo, membershipOrgUnitRepo)
	userRepo := data.NewUserRepo(ctx, entClient, userRoleRepo, userOrgUnitRepo, userPositionRepo, membershipRepo)
	rolePermissionRepo := data.NewRolePermissionRepo(ctx, entClient)
	permissionApiRepo := data.NewPermissionApiRepo(ctx, entClient)
	permissionMenuRepo := data.NewPermissionMenuRepo(ctx, entClient)
	permissionRepo := data.NewPermissionRepo(ctx, entClient, permissionApiRepo, permissionMenuRepo, rolePermissionRepo)
	roleMetadataRepo := data.NewRoleMetadataRepo(ctx, entClient)
	roleRepo := data.NewRoleRepo(ctx, entClient, rolePermissionRepo, permissionRepo, roleMetadataRepo, userRoleRepo)
	tenantRepo := data.NewTenantRepo(ctx, entClient)
	loginPolicyRepo := data.NewLoginPolicyRepo(ctx, entClient)
	taskRepo := data.NewTaskRepo(ctx, entClient)
	warehouseRepo := data.NewWarehouseRepo(ctx, entClient)
	locationRepo := data.NewLocationRepo(ctx, entClient)
	stockQuantRepo := data.NewStockQuantRepo(ctx, entClient)
	stockMoveLineRepo := data.NewStockMoveLineRepo(ctx, entClient)
	stockLotRepo := data.NewStockLotRepo(ctx, entClient)
	stockMoveRepo := data.NewStockMoveRepo(ctx, entClient)
	stockPickingRepo := data.NewStockPickingRepo(ctx, entClient, stockMoveRepo)
	accountRepo := data.NewAccountRepo(ctx, entClient)
	journalRepo := data.NewJournalRepo(ctx, entClient, accountRepo)
	purchaseOrderRepo := data.NewPurchaseOrderRepo(ctx, entClient)
	salesOrderRepo := data.NewSalesOrderRepo(ctx, entClient)
	receivableRepo := data.NewReceivableRepo(ctx, entClient)
	payableRepo := data.NewPayableRepo(ctx, entClient)
	approvalRequestRepo := data.NewApprovalRequestRepo(ctx, entClient)
	internalMessageRecipientRepo := data.NewInternalMessageRecipientRepo(ctx, entClient)
	internalMessageRepo := data.NewInternalMessageRepo(ctx, entClient, internalMessageRecipientRepo)
	approvalFlowRepo := data.NewApprovalFlowRepo(ctx, entClient)
	subscriptionRepo := data.NewSubscriptionRepo(ctx, entClient)
	planRepo := data.NewPlanRepo(ctx, entClient)
	customerRepo := data.NewCustomerRepo(ctx, entClient)
	paymentRepo := data.NewPaymentRepo(ctx, entClient)
	receiptRepo := data.NewReceiptRepo(ctx, entClient)
	supplierRepo := data.NewSupplierRepo(ctx, entClient)
	productRepo := data.NewProductRepo(ctx, entClient)
	fileRepo := data.NewFileRepo(ctx, entClient)
	dictEntryI18nRepo := data.NewDictEntryI18nRepo(ctx, entClient)
	dictEntryRepo := data.NewDictEntryRepo(ctx, entClient, dictEntryI18nRepo)
	dictTypeRepo := data.NewDictTypeRepo(ctx, entClient, dictEntryRepo)
	languageRepo := data.NewLanguageRepo(ctx, entClient)
	positionRepo := data.NewPositionRepo(ctx, entClient)
	orgUnitRepo := data.NewOrgUnitRepo(ctx, entClient)
	menuRepo := data.NewMenuRepo(ctx, entClient)
	apiRepo := data.NewApiRepo(ctx, entClient)
	permissionGroupRepo := data.NewPermissionGroupRepo(ctx, entClient)
	permissionAuditLogRepo := data.NewPermissionAuditLogRepo(ctx, entClient)
	policyEvaluationLogRepo := data.NewPolicyEvaluationLogRepo(ctx, entClient)
	loginAuditLogRepo := data.NewLoginAuditLogRepo(ctx, entClient)
	apiAuditLogRepo := data.NewApiAuditLogRepo(ctx, entClient)
	operationAuditLogRepo := data.NewOperationAuditLogRepo(ctx, entClient)
	dataAccessAuditLogRepo := data.NewDataAccessAuditLogRepo(ctx, entClient)
	internalMessageCategoryRepo := data.NewInternalMessageCategoryRepo(ctx, entClient)

	// ── register:repo ── 新模块仓储在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 三、服务层(internal/service) ═══════════════════════

	authenticationService := service.NewAuthenticationService(ctx, authenticator, userCredentialRepo, userRepo, roleRepo, tenantRepo, permissionRepo)
	loginPolicyService := service.NewLoginPolicyService(ctx, loginPolicyRepo)
	userCredentialService := service.NewUserCredentialService(ctx, userCredentialRepo)
	taskService := service.NewTaskService(ctx, taskRepo, userRepo)
	warehouseService := service.NewWarehouseService(ctx, warehouseRepo, locationRepo)
	locationService := service.NewLocationService(ctx, locationRepo)
	stockQuantService := service.NewStockQuantService(ctx, stockQuantRepo, stockMoveLineRepo, warehouseRepo)
	stockLotService := service.NewStockLotService(ctx, stockLotRepo)
	journalPoster := service.NewJournalPoster(ctx, journalRepo)
	stockPickingService := service.NewStockPickingService(ctx, stockPickingRepo, stockQuantRepo, stockMoveLineRepo, stockLotRepo, journalPoster, purchaseOrderRepo, salesOrderRepo, receivableRepo, payableRepo, approvalRequestRepo, locationRepo, internalMessageRepo, internalMessageRecipientRepo)
	billingGuard := service.NewBillingGuard(ctx, subscriptionRepo, planRepo, userRepo, purchaseOrderRepo, salesOrderRepo)
	purchaseOrderService := service.NewPurchaseOrderService(ctx, purchaseOrderRepo, approvalRequestRepo, payableRepo, billingGuard, stockQuantRepo, locationRepo, stockPickingRepo)
	salesOrderService := service.NewSalesOrderService(ctx, salesOrderRepo, approvalRequestRepo, receivableRepo, customerRepo, billingGuard, locationRepo, stockPickingRepo, journalPoster)
	paymentService := service.NewPaymentService(ctx, paymentRepo, payableRepo, approvalRequestRepo, journalPoster)
	receiptService := service.NewReceiptService(ctx, receiptRepo, receivableRepo, approvalRequestRepo, journalPoster)
	approvalRequestService := service.NewApprovalRequestService(ctx, approvalRequestRepo, approvalFlowRepo, purchaseOrderService, salesOrderService, paymentService, receiptService, internalMessageRepo, internalMessageRecipientRepo)
	approvalFlowService := service.NewApprovalFlowService(ctx, approvalFlowRepo)
	supplierService := service.NewSupplierService(ctx, supplierRepo)
	productService := service.NewProductService(ctx, productRepo)
	customerService := service.NewCustomerService(ctx, customerRepo)
	payableService := service.NewPayableService(ctx, payableRepo)
	accountingService := service.NewAccountingService(ctx, accountRepo, journalRepo)
	receivableService := service.NewReceivableService(ctx, receivableRepo)
	financeReportService := service.NewFinanceReportService(ctx, salesOrderRepo, stockPickingRepo, stockMoveLineRepo, journalRepo, receivableRepo, payableRepo, receiptRepo, paymentRepo, productRepo, customerRepo)
	billingService := service.NewBillingService(ctx, planRepo, subscriptionRepo, userRepo, purchaseOrderRepo, salesOrderRepo)
	planAdminService := service.NewPlanAdminService(ctx, planRepo)
	fileService := service.NewFileService(ctx, fileRepo, minIOClient)
	dictTypeService := service.NewDictTypeService(ctx, dictTypeRepo)
	dictEntryService := service.NewDictEntryService(ctx, dictEntryRepo)
	languageService := service.NewLanguageService(ctx, languageRepo)
	tenantService := service.NewTenantService(ctx, tenantRepo, userRepo, userCredentialRepo, roleRepo, locationRepo, warehouseService, productService, supplierService, customerService)
	loggingContactSender := service.NewLoggingContactSender(ctx)
	userService := service.NewUserService(ctx, userRepo, roleRepo, userCredentialRepo, positionRepo, orgUnitRepo, tenantRepo, membershipRepo, minIOClient, fileRepo, contactCodeCache, loggingContactSender)
	roleService := service.NewRoleService(ctx, roleRepo, tenantRepo, userRoleRepo, userRepo)
	positionService := service.NewPositionService(ctx, positionRepo, orgUnitRepo)
	orgUnitService := service.NewOrgUnitService(ctx, orgUnitRepo, userRepo)
	menuService := service.NewMenuService(ctx, menuRepo)
	apiService := service.NewApiService(ctx, apiRepo)
	permissionService := service.NewPermissionService(ctx, permissionRepo, permissionGroupRepo, menuRepo, apiRepo, roleRepo)
	permissionGroupService := service.NewPermissionGroupService(ctx, permissionGroupRepo, permissionRepo)
	permissionAuditLogService := service.NewPermissionAuditLogService(ctx, permissionAuditLogRepo)
	policyEvaluationLogService := service.NewPolicyEvaluationLogService(ctx, policyEvaluationLogRepo)
	loginAuditLogService := service.NewLoginAuditLogService(ctx, loginAuditLogRepo)
	apiAuditLogService := service.NewApiAuditLogService(ctx, apiAuditLogRepo, apiRepo)
	operationAuditLogService := service.NewOperationAuditLogService(ctx, operationAuditLogRepo)
	dataAccessAuditLogService := service.NewDataAccessAuditLogService(ctx, dataAccessAuditLogRepo)
	internalMessageService := service.NewInternalMessageService(ctx, internalMessageRepo, internalMessageCategoryRepo, internalMessageRecipientRepo, userRepo)
	internalMessageCategoryService := service.NewInternalMessageCategoryService(ctx, internalMessageCategoryRepo)
	internalMessageRecipientService := service.NewInternalMessageRecipientService(ctx, internalMessageRepo, internalMessageRecipientRepo)
	demoDataService := service.NewDemoDataService(ctx, tenantService, warehouseService, productService, supplierService, stockQuantRepo, locationRepo)

	// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)

	// ═══════════════════════ 四、传输层(internal/server) ═══════════════════════

	grpcMiddlewares := server.NewGrpcMiddleware(ctx)

	grpcServer, err := server.NewGrpcServer(ctx, grpcMiddlewares,
		authenticationService,
		loginPolicyService,
		userCredentialService,
		taskService,
		warehouseService,
		locationService,
		stockQuantService,
		stockLotService,
		stockPickingService,
		approvalRequestService,
		approvalFlowService,
		supplierService,
		productService,
		purchaseOrderService,
		customerService,
		salesOrderService,
		payableService,
		accountingService,
		paymentService,
		receivableService,
		receiptService,
		financeReportService,
		billingService,
		planAdminService,
		fileService,
		dictTypeService,
		dictEntryService,
		languageService,
		tenantService,
		userService,
		roleService,
		positionService,
		orgUnitService,
		menuService,
		apiService,
		permissionService,
		permissionGroupService,
		permissionAuditLogService,
		policyEvaluationLogService,
		loginAuditLogService,
		apiAuditLogService,
		operationAuditLogService,
		dataAccessAuditLogService,
		internalMessageService,
		internalMessageCategoryService,
		internalMessageRecipientService,
		demoDataService,
		// register:grpc-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)
	)
	if err != nil {
		rollback()
		return nil, nil, err
	}

	asynqServer := server.NewAsynqServer(ctx, taskService)

	return newApp(ctx, grpcServer, asynqServer), rollback, nil
}
