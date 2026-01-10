package observability

import (
	"context"

	"github.com/google/uuid"
)

// Context keys for observability data.
type contextKey string

const (
	correlationIDCtxKey contextKey = "correlation_id"
	requestIDCtxKey     contextKey = "request_id"
	userIDCtxKey        contextKey = "user_id"
	operationCtxKey     contextKey = "operation"
)

// Standard attribute keys used in logs and metrics.
const (
	CorrelationIDKey = "correlation_id"
	RequestIDKey     = "request_id"
	UserIDKey        = "user_id"
	OperationKey     = "operation"
	DurationKey      = "duration_ms"
	ErrorKey         = "error"
	StatusKey        = "status"
)

// WithCorrelationID adds a correlation ID to the context.
// If id is empty, a new UUID is generated.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, correlationIDCtxKey, id)
}

// CorrelationIDFromContext extracts the correlation ID from context.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDCtxKey).(string); ok {
		return id
	}
	return ""
}

// WithRequestID adds a request ID to the context.
// If id is empty, a new UUID is generated.
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, requestIDCtxKey, id)
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDCtxKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDCtxKey, userID)
}

// UserIDFromContext extracts the user ID from context.
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(userIDCtxKey).(string); ok {
		return id
	}
	return ""
}

// WithOperation adds an operation name to the context.
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationCtxKey, operation)
}

// OperationFromContext extracts the operation name from context.
func OperationFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if op, ok := ctx.Value(operationCtxKey).(string); ok {
		return op
	}
	return ""
}

// NewRequestContext creates a context with a new request ID and correlation ID.
// If parentCorrelationID is provided, it's used; otherwise a new one is generated.
func NewRequestContext(ctx context.Context, parentCorrelationID string) context.Context {
	ctx = WithRequestID(ctx, "")
	if parentCorrelationID != "" {
		ctx = WithCorrelationID(ctx, parentCorrelationID)
	} else {
		ctx = WithCorrelationID(ctx, "")
	}
	return ctx
}
