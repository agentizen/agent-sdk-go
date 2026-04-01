package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultMaxResponseBytes int64         = 10 * 1024 * 1024 // 10MB
	defaultTimeout          time.Duration = 30 * time.Second
)

// Client is the transport interface for communicating with MCP servers.
// The SDK provides HTTPClient as a default implementation. Consumers can
// implement gRPC, WebSocket, stdio, or any other transport.
type Client interface {
	// Execute invokes a named tool on the server with the given parameters.
	Execute(ctx context.Context, server ServerConfig, toolName string, params map[string]interface{}) (interface{}, error)

	// ListTools returns the tools exposed by the server.
	ListTools(ctx context.Context, server ServerConfig) ([]ToolInfo, error)
}

// ToolInfo describes a tool exposed by an MCP server.
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// HTTPClient implements Client using HTTP transport.
//
// Execute sends a POST to server.URL with a JSON body containing
// "tool" (the tool name) and "params" (the parameters).
//
// ListTools sends a POST to server.URL with a JSON body containing
// "action": "list_tools".
//
// The URL is used as-is — the client does not append any path segments.
// Configure the full endpoint URL in ServerConfig.URL.
type HTTPClient struct {
	httpClient *http.Client
	opts       ClientOptions
}

// NewHTTPClient creates an HTTPClient with the given options.
// Zero-value options fields are replaced with sensible defaults.
func NewHTTPClient(opts ClientOptions) *HTTPClient {
	if opts.MaxResponseBytes == 0 {
		opts.MaxResponseBytes = defaultMaxResponseBytes
	}
	if opts.DefaultTimeout == 0 {
		opts.DefaultTimeout = defaultTimeout
	}
	return &HTTPClient{
		httpClient: &http.Client{},
		opts:       opts,
	}
}

// SetHTTPClient replaces the underlying http.Client. This is useful in tests
// to inject a client that trusts a test server's TLS certificate.
func (c *HTTPClient) SetHTTPClient(hc *http.Client) {
	c.httpClient = hc
}

// executeRequest is the request payload sent to MCP servers.
type executeRequest struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

// listRequest is the request payload for tool discovery.
type listRequest struct {
	Action string `json:"action"`
}

// Execute sends a POST to server.URL with a JSON body containing the tool
// name and parameters. The URL is used as-is without modification.
func (c *HTTPClient) Execute(ctx context.Context, server ServerConfig, toolName string, params map[string]interface{}) (interface{}, error) {
	if err := c.validateURL(server.URL); err != nil {
		return nil, err
	}

	payload := executeRequest{
		Tool:   toolName,
		Params: params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to marshal request: %w", err)
	}

	ctx, cancel := c.withTimeout(ctx, server.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyHeaders(req, server)

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("mcp: failed to decode response: %w", err)
	}
	return result, nil
}

// ListTools sends a POST to server.URL with an action payload requesting
// the list of available tools. The URL is used as-is without modification.
func (c *HTTPClient) ListTools(ctx context.Context, server ServerConfig) ([]ToolInfo, error) {
	if err := c.validateURL(server.URL); err != nil {
		return nil, err
	}

	body, err := json.Marshal(listRequest{Action: "list_tools"})
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to marshal request: %w", err)
	}

	ctx, cancel := c.withTimeout(ctx, server.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyHeaders(req, server)

	respBody, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	var tools []ToolInfo
	if err := json.Unmarshal(respBody, &tools); err != nil {
		return nil, fmt.Errorf("mcp: failed to decode tools: %w", err)
	}
	return tools, nil
}

// withTimeout creates a context with the server-specific or default timeout.
func (c *HTTPClient) withTimeout(ctx context.Context, serverTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := serverTimeout
	if timeout == 0 {
		timeout = c.opts.DefaultTimeout
	}
	return context.WithTimeout(ctx, timeout)
}

// doRequest executes the HTTP request, enforces the response size limit,
// and checks the status code.
func (c *HTTPClient) doRequest(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mcp: request failed: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.opts.MaxResponseBytes)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp: server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// validateURL rejects http:// URLs unless AllowHTTP is set.
func (c *HTTPClient) validateURL(rawURL string) error {
	if !c.opts.AllowHTTP && strings.HasPrefix(rawURL, "http://") {
		return fmt.Errorf("mcp: http:// URLs are not allowed (set AllowHTTP to enable)")
	}
	return nil
}

// applyHeaders sets server headers, context headers, and X-User-ID on the request.
func (c *HTTPClient) applyHeaders(req *http.Request, server ServerConfig) {
	for k, v := range server.Headers {
		req.Header.Set(k, v)
	}
	if ctxHeaders := HeadersFromContext(req.Context()); ctxHeaders != nil {
		for k, v := range ctxHeaders {
			req.Header.Set(k, v)
		}
	}
	if userID := UserIDFromContext(req.Context()); userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
}
