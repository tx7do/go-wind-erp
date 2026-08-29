package constants

const (
	// RoleCodeSpilt 角色代码分隔符
	RoleCodeSpilt = ":"

	// TemplateRoleCodePrefix 模板角色代码前缀
	TemplateRoleCodePrefix = "template" + RoleCodeSpilt
	// PlatformRoleCodePrefix 平台角色代码前缀
	PlatformRoleCodePrefix = "platform" + RoleCodeSpilt
	// TenantRoleCodePrefix 租户角色代码前缀
	TenantRoleCodePrefix = "tenant" + RoleCodeSpilt

	// PlatformAdminRoleCode 平台管理员角色代码
	PlatformAdminRoleCode = PlatformRoleCodePrefix + "admin"
	// TenantAdminRoleCode 租户管理员角色代码
	TenantAdminRoleCode = TenantRoleCodePrefix + "manager"
	// TenantAdminTemplateRoleCode 租户管理员模板角色代码
	TenantAdminTemplateRoleCode = TemplateRoleCodePrefix + TenantAdminRoleCode

	// DefaultPlatformAdminRoleName 平台管理员角色默认名称
	DefaultPlatformAdminRoleName = "平台管理员"
	// DefaultTenantManagerRoleName 租户管理员角色默认名称
	DefaultTenantManagerRoleName = "租户管理员"
)

func HasRoleCodePrefix(roleCode string, prefix string) bool {
	return len(roleCode) >= len(prefix) && roleCode[0:len(prefix)] == prefix
}

func IsPlatformRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, PlatformRoleCodePrefix)
}

func IsTenantRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, TenantRoleCodePrefix)
}

func IsTemplateRoleCode(roleCode string) bool {
	return HasRoleCodePrefix(roleCode, TemplateRoleCodePrefix)
}

func ExtractRoleCodeFromTemplate(roleCode string) string {
	if IsTemplateRoleCode(roleCode) {
		return roleCode[len(TemplateRoleCodePrefix):]
	}
	return roleCode
}

// 套餐编码（透明定价，sub_plans.code 的合法取值）
const (
	FreePlanCode     = "FREE"
	StandardPlanCode = "STANDARD"
	ProPlanCode      = "PRO"
)

// 会计科目编码（简易总账，fin_accounts.code 的固定取值）
const (
	AccountCodeCash        = "1001" // 库存现金
	AccountCodeBank        = "1002" // 银行存款
	AccountCodeAR          = "1122" // 应收账款
	AccountCodeInventory   = "1405" // 库存商品
	AccountCodePendingLoss = "1901" // 待处理财产损溢（盘点损益）
	AccountCodeAP          = "2202" // 应付账款
	AccountCodeRevenue     = "6001" // 主营业务收入
	AccountCodeCOGS        = "6401" // 主营业务成本
)
