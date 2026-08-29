package server

import (
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/transport/grpc"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/rpc"

	"go-wind-erp/app/core/service/internal/service"

	approvalV1 "go-wind-erp/api/gen/go/approval/service/v1"
	adminV1 "go-wind-erp/api/gen/go/admin/service/v1"
	financeV1 "go-wind-erp/api/gen/go/finance/service/v1"
	procurementV1 "go-wind-erp/api/gen/go/procurement/service/v1"
	productV1 "go-wind-erp/api/gen/go/product/service/v1"
	salesV1 "go-wind-erp/api/gen/go/sales/service/v1"
	auditV1 "go-wind-erp/api/gen/go/audit/service/v1"
	authenticationV1 "go-wind-erp/api/gen/go/authentication/service/v1"
	billingV1 "go-wind-erp/api/gen/go/billing/service/v1"
	dictV1 "go-wind-erp/api/gen/go/dict/service/v1"
	identityV1 "go-wind-erp/api/gen/go/identity/service/v1"
	internalMessageV1 "go-wind-erp/api/gen/go/internal_message/service/v1"
	inventoryV1 "go-wind-erp/api/gen/go/inventory/service/v1"
	permissionV1 "go-wind-erp/api/gen/go/permission/service/v1"
	storageV1 "go-wind-erp/api/gen/go/storage/service/v1"
	taskV1 "go-wind-erp/api/gen/go/task/service/v1"

	"go-wind-erp/pkg/middleware/ent"
)

func NewGrpcMiddleware(ctx *bootstrap.Context) []middleware.Middleware {
	var ms []middleware.Middleware
	ms = append(ms, logging.Server(ctx.GetLogger()))
	ms = append(ms, ent.Server())
	return ms
}

// NewGrpcServer new a gRPC server.
func NewGrpcServer(
	ctx *bootstrap.Context,
	middlewares []middleware.Middleware,

	authenticationService *service.AuthenticationService,
	loginPolicyService *service.LoginPolicyService,
	userCredentialService *service.UserCredentialService,

	taskService *service.TaskService,

	warehouseService *service.WarehouseService,
	locationService *service.LocationService,
	stockQuantService *service.StockQuantService,
	stockLotService   *service.StockLotService,
	stockPickingService *service.StockPickingService,

	approvalRequestService *service.ApprovalRequestService,
	approvalFlowService   *service.ApprovalFlowService,

	supplierService *service.SupplierService,
	productService *service.ProductService,
	purchaseOrderService *service.PurchaseOrderService,

	customerService *service.CustomerService,
	salesOrderService *service.SalesOrderService,

	payableService *service.PayableService,
	accountingService *service.AccountingService,
	paymentService *service.PaymentService,
	receivableService *service.ReceivableService,
	receiptService *service.ReceiptService,
	financeReportService *service.FinanceReportService,

	billingService *service.BillingService,
	planAdminService *service.PlanAdminService,

	fileService *service.FileService,

	dictTypeService *service.DictTypeService,
	dictEntryService *service.DictEntryService,
	languageService *service.LanguageService,

	tenantService *service.TenantService,
	userService *service.UserService,
	roleService *service.RoleService,
	positionService *service.PositionService,
	orgUnitService *service.OrgUnitService,

	menuService *service.MenuService,
	apiService *service.ApiService,
	permissionService *service.PermissionService,
	permissionGroupService *service.PermissionGroupService,
	permissionAuditLogService *service.PermissionAuditLogService,
	policyEvaluationLogService *service.PolicyEvaluationLogService,

	loginAuditLogService *service.LoginAuditLogService,
	apiAuditLogService *service.ApiAuditLogService,
	operationAuditLogService *service.OperationAuditLogService,
	dataAccessAuditLogService *service.DataAccessAuditLogService,

	internalMessageService *service.InternalMessageService,
	internalMessageCategoryService *service.InternalMessageCategoryService,
	internalMessageRecipientService *service.InternalMessageRecipientService,

	_ *service.DemoDataService,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}

	taskV1.RegisterTaskServiceServer(srv, taskService)

	inventoryV1.RegisterWarehouseServiceServer(srv, warehouseService)
	inventoryV1.RegisterLocationServiceServer(srv, locationService)
	inventoryV1.RegisterStockQuantServiceServer(srv, stockQuantService)
	inventoryV1.RegisterStockLotServiceServer(srv, stockLotService)
	inventoryV1.RegisterStockPickingServiceServer(srv, stockPickingService)

	approvalV1.RegisterApprovalRequestServiceServer(srv, approvalRequestService)
	approvalV1.RegisterApprovalFlowServiceServer(srv, approvalFlowService)

	procurementV1.RegisterSupplierServiceServer(srv, supplierService)
	productV1.RegisterProductServiceServer(srv, productService)
	procurementV1.RegisterPurchaseOrderServiceServer(srv, purchaseOrderService)

	salesV1.RegisterCustomerServiceServer(srv, customerService)
	salesV1.RegisterSalesOrderServiceServer(srv, salesOrderService)

	financeV1.RegisterPayableServiceServer(srv, payableService)
	financeV1.RegisterAccountingServiceServer(srv, accountingService)
	financeV1.RegisterPaymentServiceServer(srv, paymentService)
	financeV1.RegisterReceivableServiceServer(srv, receivableService)
	financeV1.RegisterReceiptServiceServer(srv, receiptService)
	adminV1.RegisterFinanceReportServiceServer(srv, financeReportService)

	billingV1.RegisterBillingServiceServer(srv, billingService)
	billingV1.RegisterPlanAdminServiceServer(srv, planAdminService)

	authenticationV1.RegisterLoginPolicyServiceServer(srv, loginPolicyService)
	authenticationV1.RegisterAuthenticationServiceServer(srv, authenticationService)
	authenticationV1.RegisterUserCredentialServiceServer(srv, userCredentialService)

	dictV1.RegisterDictTypeServiceServer(srv, dictTypeService)
	dictV1.RegisterDictEntryServiceServer(srv, dictEntryService)
	dictV1.RegisterLanguageServiceServer(srv, languageService)

	permissionV1.RegisterApiServiceServer(srv, apiService)
	permissionV1.RegisterMenuServiceServer(srv, menuService)

	permissionV1.RegisterPermissionServiceServer(srv, permissionService)
	permissionV1.RegisterPermissionGroupServiceServer(srv, permissionGroupService)
	permissionV1.RegisterPolicyEvaluationLogServiceServer(srv, policyEvaluationLogService)
	permissionV1.RegisterRoleServiceServer(srv, roleService)

	identityV1.RegisterUserServiceServer(srv, userService)
	identityV1.RegisterOrgUnitServiceServer(srv, orgUnitService)
	identityV1.RegisterPositionServiceServer(srv, positionService)
	identityV1.RegisterTenantServiceServer(srv, tenantService)

	auditV1.RegisterLoginAuditLogServiceServer(srv, loginAuditLogService)
	auditV1.RegisterApiAuditLogServiceServer(srv, apiAuditLogService)
	auditV1.RegisterOperationAuditLogServiceServer(srv, operationAuditLogService)
	auditV1.RegisterDataAccessAuditLogServiceServer(srv, dataAccessAuditLogService)
	auditV1.RegisterPermissionAuditLogServiceServer(srv, permissionAuditLogService)

	storageV1.RegisterFileServiceServer(srv, fileService)

	internalMessageV1.RegisterInternalMessageServiceServer(srv, internalMessageService)
	internalMessageV1.RegisterInternalMessageCategoryServiceServer(srv, internalMessageCategoryService)
	internalMessageV1.RegisterInternalMessageRecipientServiceServer(srv, internalMessageRecipientService)

	return srv, nil
}
