package mcp_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/stretchr/testify/assert"
)

func TestWithUserID_RoundTrip(t *testing.T) {
	ctx := mcp.WithUserID(context.Background(), "user-42")
	assert.Equal(t, "user-42", mcp.UserIDFromContext(ctx))
}

func TestUserIDFromContext_EmptyContext(t *testing.T) {
	assert.Equal(t, "", mcp.UserIDFromContext(context.Background()))
}

func TestWithHeaders_RoundTrip(t *testing.T) {
	headers := map[string]string{"X-Tenant": "acme", "Authorization": "Bearer tok"}
	ctx := mcp.WithHeaders(context.Background(), headers)
	got := mcp.HeadersFromContext(ctx)
	assert.Equal(t, headers, got)
}

func TestHeadersFromContext_EmptyContext(t *testing.T) {
	assert.Nil(t, mcp.HeadersFromContext(context.Background()))
}

func TestWithUserIDAndHeaders_Combined(t *testing.T) {
	ctx := context.Background()
	ctx = mcp.WithUserID(ctx, "user-99")
	ctx = mcp.WithHeaders(ctx, map[string]string{"X-Foo": "bar"})

	assert.Equal(t, "user-99", mcp.UserIDFromContext(ctx))
	assert.Equal(t, map[string]string{"X-Foo": "bar"}, mcp.HeadersFromContext(ctx))
}
