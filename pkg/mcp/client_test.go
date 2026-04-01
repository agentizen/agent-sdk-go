package mcp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_Execute_PostsToServerURL(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	params := map[string]interface{}{"query": "hello"}
	result, err := client.Execute(context.Background(), cfg, "search", params)
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	// URL is used as-is — no path segments appended
	assert.Equal(t, "/", gotPath)
	// Body contains tool name and params
	assert.Equal(t, "search", gotBody["tool"])
	paramsMap, ok := gotBody["params"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello", paramsMap["query"])

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, resultMap["ok"])
}

func TestHTTPClient_Execute_UsesCustomPath(t *testing.T) {
	var gotPath string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL + "/api/v1/mcp",
		Client: client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/mcp", gotPath)
}

func TestHTTPClient_ListTools_PostsAndParsesResponse(t *testing.T) {
	var gotMethod string
	var gotBody map[string]interface{}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"calc","description":"Calculator","parameters":{"type":"object"}}]`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	tools, err := client.ListTools(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "list_tools", gotBody["action"])
	require.Len(t, tools, 1)
	assert.Equal(t, "calc", tools[0].Name)
	assert.Equal(t, "Calculator", tools[0].Description)
}

func TestHTTPClient_ServerHeaders_AreIncluded(t *testing.T) {
	var gotHeaders http.Header

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle:  "test",
		URL:     srv.URL,
		Headers: map[string]string{"X-Api-Key": "secret123"},
		Client:  client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.NoError(t, err)
	assert.Equal(t, "secret123", gotHeaders.Get("X-Api-Key"))
}

func TestHTTPClient_ContextHeaders_AreIncluded(t *testing.T) {
	var gotHeaders http.Header

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	ctx := mcp.WithHeaders(context.Background(), map[string]string{"X-Trace-Id": "abc-123"})
	_, err := client.Execute(ctx, cfg, "noop", nil)
	require.NoError(t, err)
	assert.Equal(t, "abc-123", gotHeaders.Get("X-Trace-Id"))
}

func TestHTTPClient_UserIDHeader_IsIncluded(t *testing.T) {
	var gotHeaders http.Header

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	ctx := mcp.WithUserID(context.Background(), "user-42")
	_, err := client.Execute(ctx, cfg, "noop", nil)
	require.NoError(t, err)
	assert.Equal(t, "user-42", gotHeaders.Get("X-User-ID"))
}

func TestHTTPClient_RejectsHTTP_WhenAllowHTTPFalse(t *testing.T) {
	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: false})
	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    "http://insecure.example.com",
		Client: client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http:// URLs are not allowed")

	_, err = client.ListTools(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http:// URLs are not allowed")
}

func TestHTTPClient_AllowsHTTP_WhenAllowHTTPTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	assert.NoError(t, err)
}

func TestHTTPClient_ResponseBodySizeLimit(t *testing.T) {
	largeBody := strings.Repeat("x", 200)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"` + largeBody + `"`))
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{
		AllowHTTP:        true,
		MaxResponseBytes: 50,
	})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestHTTPClient_ServerErrorStatus(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{
		Handle: "test",
		URL:    srv.URL,
		Client: client,
	}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}
