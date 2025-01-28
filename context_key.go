package kit

// ContextKey defines custom keys for storing values in the context.
type ContextKey string

// Predefined context keys
const (
	Logger              ContextKey = "kit.logger"
	UserEmail           ContextKey = "kit.user_email"
	UserCompany         ContextKey = "kit.user_company"
	UserCompanyCategory ContextKey = "kit.user_company_category"
	UserPermissions     ContextKey = "kit.user_permissions"
)
