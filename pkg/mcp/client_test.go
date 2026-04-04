package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers ---

// jsonrpcRequest mirrors the wire format for assertions.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse is the wire format for test server responses.
type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// mcpTestHandler is a simplified MCP server handler for tests.
// It auto-handles initialize and notifications/initialized,
// and delegates tools/* methods to the provided callback.
type mcpTestHandler struct {
	sessionID string
	onMethod  func(method string, params json.RawMessage) (interface{}, error)
}

func (h *mcpTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req jsonrpcRequest
	_ = json.Unmarshal(body, &req)

	// Set session ID on all responses
	if h.sessionID != "" {
		w.Header().Set("Mcp-Session-Id", h.sessionID)
	}

	// Handle notifications (no ID)
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var result interface{}
	var rpcErr interface{}

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "test-server", "version": "1.0.0"},
		}
	default:
		if h.onMethod != nil {
			var err error
			result, err = h.onMethod(req.Method, req.Params)
			if err != nil {
				rpcErr = map[string]interface{}{"code": -32000, "message": err.Error()}
			}
		}
	}

	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
		Error:   rpcErr,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func newMCPTestServer(t *testing.T, handler *mcpTestHandler) (*httptest.Server, *mcp.HTTPClient) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())
	return srv, client
}

// --- Tests ---

func TestHTTPClient_Execute_SendsJSONRPC_ToolsCall(t *testing.T) {
	var gotMethod string
	var gotParams json.RawMessage

	handler := &mcpTestHandler{
		onMethod: func(method string, params json.RawMessage) (interface{}, error) {
			gotMethod = method
			gotParams = params
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "result-value"},
				},
				"isError": false,
			}, nil
		},
	}
	srv, client := newMCPTestServer(t, handler)

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	result, err := client.Execute(context.Background(), cfg, "search", map[string]interface{}{"query": "hello"})
	require.NoError(t, err)

	assert.Equal(t, "tools/call", gotMethod)

	var params map[string]interface{}
	require.NoError(t, json.Unmarshal(gotParams, &params))
	assert.Equal(t, "search", params["name"])
	args, ok := params["arguments"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello", args["query"])

	assert.Equal(t, "result-value", result)
}

func TestHTTPClient_Execute_UsesCustomPath(t *testing.T) {
	var gotPath string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		w.Header().Set("Content-Type", "application/json")

		var result interface{}
		switch req.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"capabilities":    map[string]interface{}{},
				"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
			}
		default:
			result = map[string]interface{}{
				"content": []map[string]interface{}{{"type": "text", "text": "ok"}},
				"isError": false,
			}
		}

		// notifications have no ID
		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL + "/api/v1/mcp", Client: client}
	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/mcp", gotPath)
}

func TestHTTPClient_ListTools_SendsJSONRPC_ToolsList(t *testing.T) {
	var gotMethod string

	handler := &mcpTestHandler{
		onMethod: func(method string, _ json.RawMessage) (interface{}, error) {
			gotMethod = method
			return map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "calc",
						"description": "Calculator",
						"inputSchema": map[string]interface{}{"type": "object"},
					},
				},
			}, nil
		},
	}
	srv, client := newMCPTestServer(t, handler)

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	tools, err := client.ListTools(context.Background(), cfg)
	require.NoError(t, err)

	assert.Equal(t, "tools/list", gotMethod)
	require.Len(t, tools, 1)
	assert.Equal(t, "calc", tools[0].Name)
	assert.Equal(t, "Calculator", tools[0].Description)
	assert.Equal(t, map[string]interface{}{"type": "object"}, tools[0].Parameters)
}

func TestHTTPClient_AcceptHeader_IsSet(t *testing.T) {
	var gotAccept string

	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{"tools": []interface{}{}}, nil
		},
	}

	// Use raw httptest to capture headers from the tools/list call
	var callCount atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		// Capture Accept header from every request
		gotAccept = r.Header.Get("Accept")
		handler.ServeHTTP(w, r)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	_, _ = client.ListTools(context.Background(), cfg)

	assert.Equal(t, "application/json, text/event-stream", gotAccept)
}

func TestHTTPClient_InitializeHandshake_HappensOnce(t *testing.T) {
	var methods []string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		methods = append(methods, req.Method)

		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		var result interface{}
		switch req.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"capabilities":    map[string]interface{}{},
				"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
			}
		default:
			result = map[string]interface{}{"tools": []interface{}{}}
		}
		resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())
	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}

	// First call triggers initialize handshake
	_, err := client.ListTools(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"initialize", "notifications/initialized", "tools/list"}, methods)

	// Second call reuses session — no re-initialize
	methods = nil
	_, err = client.ListTools(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"tools/list"}, methods)
}

func TestHTTPClient_SessionID_IsPropagated(t *testing.T) {
	var lastSessionID string

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastSessionID = r.Header.Get("Mcp-Session-Id")

		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		// Always set session ID on response
		w.Header().Set("Mcp-Session-Id", "sess-abc-123")

		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		var result interface{}
		switch req.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": "2025-03-26",
				"capabilities":    map[string]interface{}{},
				"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
			}
		default:
			result = map[string]interface{}{"tools": []interface{}{}}
		}
		resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())
	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}

	// First call: initialize + tools/list
	_, err := client.ListTools(context.Background(), cfg)
	require.NoError(t, err)

	// Second call should include the session ID from the first response
	_, err = client.ListTools(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "sess-abc-123", lastSessionID)
}

func TestHTTPClient_ServerHeaders_AreIncluded(t *testing.T) {
	var gotHeaders http.Header

	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{"tools": []interface{}{}}, nil
		},
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		handler.ServeHTTP(w, r)
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

	_, err := client.ListTools(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, "secret123", gotHeaders.Get("X-Api-Key"))
}

func TestHTTPClient_ContextHeaders_AreIncluded(t *testing.T) {
	var gotHeaders http.Header

	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{"tools": []interface{}{}}, nil
		},
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		handler.ServeHTTP(w, r)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	ctx := mcp.WithHeaders(context.Background(), map[string]string{"X-Trace-Id": "abc-123"})

	_, err := client.ListTools(ctx, cfg)
	require.NoError(t, err)
	assert.Equal(t, "abc-123", gotHeaders.Get("X-Trace-Id"))
}

func TestHTTPClient_UserIDHeader_IsIncluded(t *testing.T) {
	var gotHeaders http.Header

	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{"tools": []interface{}{}}, nil
		},
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		handler.ServeHTTP(w, r)
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	ctx := mcp.WithUserID(context.Background(), "user-42")

	_, err := client.ListTools(ctx, cfg)
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
	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{"tools": []interface{}{}}, nil
		},
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}

	_, err := client.ListTools(context.Background(), cfg)
	assert.NoError(t, err)
}

func TestHTTPClient_ResponseBodySizeLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "initialize":
			resp := jsonrpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]interface{}{},
					"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			// Return a very large response
			largeText := strings.Repeat("x", 500)
			resp := jsonrpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{{"type": "text", "text": largeText}},
					"isError": false,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{
		AllowHTTP:        true,
		MaxResponseBytes: 50,
	})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}

	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestHTTPClient_ServerErrorStatus(t *testing.T) {
	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return nil, fmt.Errorf("internal error")
		},
	}
	srv, client := newMCPTestServer(t, handler)

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	_, err := client.Execute(context.Background(), cfg, "noop", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
}

func TestHTTPClient_ToolError_IsReturned(t *testing.T) {
	handler := &mcpTestHandler{
		onMethod: func(_ string, _ json.RawMessage) (interface{}, error) {
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "City not found: Atlantis"},
				},
				"isError": true,
			}, nil
		},
	}
	srv, client := newMCPTestServer(t, handler)

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	_, err := client.Execute(context.Background(), cfg, "search_city", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool error")
	assert.Contains(t, err.Error(), "City not found: Atlantis")
}

func TestHTTPClient_SSEResponse_IsParsed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		switch req.Method {
		case "initialize":
			w.Header().Set("Content-Type", "application/json")
			resp := jsonrpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]interface{}{},
					"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			// Respond with SSE format
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			resultJSON, _ := json.Marshal(jsonrpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": "SSE result"},
					},
					"isError": false,
				},
			})

			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(resultJSON))
		}
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	result, err := client.Execute(context.Background(), cfg, "sse_tool", nil)
	require.NoError(t, err)
	assert.Equal(t, "SSE result", result)
}

func TestHTTPClient_JSONRPCError_IsReturned(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req jsonrpcRequest
		_ = json.Unmarshal(body, &req)

		if req.ID == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch req.Method {
		case "initialize":
			resp := jsonrpcResponse{
				JSONRPC: "2.0", ID: req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2025-03-26",
					"capabilities":    map[string]interface{}{},
					"serverInfo":      map[string]interface{}{"name": "test", "version": "1.0.0"},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			// Return a JSON-RPC error
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "Method not found"},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	client := mcp.NewHTTPClient(mcp.ClientOptions{AllowHTTP: true})
	client.SetHTTPClient(srv.Client())

	cfg := mcp.ServerConfig{Handle: "test", URL: srv.URL, Client: client}
	_, err := client.Execute(context.Background(), cfg, "unknown_tool", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Method not found")
}
