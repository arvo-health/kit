// Package kit provides foundational utilities for building structured Go applications.
// This file defines custom keys for storing values in the context.

package kit

// ContextKey represents a key used for storing values in the context.
type ContextKey string

// Predefined context keys for common use cases.
const (
	CtxKeyLogger              ContextKey = "kit.logger"
	CtxKeyRequestID           ContextKey = "kit.request_id"
	CtxKeyUserEmail           ContextKey = "kit.user_email"
	CtxKeyUserCompany         ContextKey = "kit.user_company"
	CtxKeyUserCompanyCategory ContextKey = "kit.user_company_category"
	CtxKeyUserPermissions     ContextKey = "kit.user_permissions"
)
