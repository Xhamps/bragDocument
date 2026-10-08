// Package telemetry provides logging and metrics plumbing shared by cmd and
// adapters.
package telemetry

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	tenantIDKey
)

// WithRequestID stores the request id in ctx.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID returns the request id stored in ctx, or "".
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithTenantID stores the tenant id in ctx.
func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

// TenantID returns the tenant id stored in ctx, or "".
func TenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}
