//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

// This file defines the dependency injection ProviderSet for the data layer and contains no business logic.
// The build tag `wireinject` excludes this source from normal `go build` and final binaries.
// Run `go generate ./...` or `go run github.com/google/wire/cmd/wire` to regenerate the Wire output (e.g. `wire_gen.go`), which will be included in final builds.
// Keep provider constructors here only; avoid init-time side effects or runtime logic in this file.

package providers

import (
	"github.com/google/wire"

	"go-wind-erp/app/admin/service/internal/service"
)

// ProviderSet is the Wire provider set for data layer.
var ProviderSet = wire.NewSet(
	service.NewAuthenticationService,
	service.NewLoginPolicyService,

	service.NewUserService,
	service.NewRoleService,
	service.NewTenantService,
	service.NewUserProfileService,
	service.NewPositionService,
	service.NewOrgUnitService,

	service.NewMenuService,
	service.NewApiService,
	service.NewPermissionGroupService,
	service.NewPermissionService,

	service.NewRouterService,

	service.NewTaskService,

	service.NewWarehouseService,
	service.NewLocationService,
	service.NewStockQuantService,
	service.NewStockPickingService,

	service.NewApprovalRequestService,

	service.NewSupplierService,

	service.NewProductService,
	service.NewPurchaseOrderService,

	service.NewPayableService,
	service.NewPaymentService,
	service.NewReceivableService,
	service.NewReceiptService,
	service.NewFinanceReportService,

	service.NewCustomerService,
	service.NewSalesOrderService,
	service.NewBillingService,
	service.NewPlanAdminService,

	service.NewFileTransferService,
	service.NewFileService,

	service.NewDictTypeService,
	service.NewDictEntryService,
	service.NewLanguageService,

	service.NewApiAuditLogService,
	service.NewDataAccessAuditLogService,
	service.NewLoginAuditLogService,
	service.NewOperationAuditLogService,
	service.NewPermissionAuditLogService,
	service.NewPolicyEvaluationLogService,

	service.NewInternalMessageService,
	service.NewInternalMessageCategoryService,
	service.NewInternalMessageRecipientService,
)
