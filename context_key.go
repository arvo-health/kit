// Package kit provides foundational utilities for building structured Go applications.
// This file defines custom keys for storing values in the context.

package kit

// ContextKey represents a key used for storing values in the context.
type ContextKey string

// Predefined context keys for common use cases.
const (
	KeyLogger              ContextKey = "kit.logger"
	KeyUserEmail           ContextKey = "kit.user_email"
	KeyUserCompany         ContextKey = "kit.user_company"
	KeyUserCompanyCategory ContextKey = "kit.user_company_category"
	KeyUserPermissions     ContextKey = "kit.user_permissions"
)
