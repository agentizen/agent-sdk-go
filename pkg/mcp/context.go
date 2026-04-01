package mcp

import "context"

type contextKey int

const (
	userIDKey  contextKey = iota
	headersKey contextKey = iota
)

// WithUserID returns a new context carrying the given user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context.
// Returns an empty string if no user ID is present.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// WithHeaders returns a new context carrying the given headers map.
func WithHeaders(ctx context.Context, headers map[string]string) context.Context {
	return context.WithValue(ctx, headersKey, headers)
}

// HeadersFromContext extracts the headers map from the context.
// Returns nil if no headers are present.
func HeadersFromContext(ctx context.Context) map[string]string {
	v, _ := ctx.Value(headersKey).(map[string]string)
	return v
}
