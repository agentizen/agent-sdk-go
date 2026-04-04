package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultMaxResponseBytes int64         = 10 * 1024 * 1024 // 10MB
	defaultTimeout          time.Duration = 30 * time.Second
	mcpProtocolVersion                    = "2025-03-26"
	mcpAcceptHeader                       = "application/json, text/event-stream"
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
	Parameters  map[string]interface{} `json:"inputSchema"` // MCP spec uses inputSchema
}

// HTTPClient implements Client using the MCP Streamable HTTP transport
// (JSON-RPC 2.0 over HTTP, protocol version 2025-03-26).
//
// On the first call to a server, HTTPClient performs the MCP initialize
// handshake (initialize + notifications/initialized). Subsequent calls
// reuse the established session.
//
// The client handles both JSON and SSE (Server-Sent Events) response
// formats, as required by the MCP specification.
type HTTPClient struct {
	httpClient *http.Client
	opts       ClientOptions
	sessions   sync.Map     // map[serverURL]*session
	nextID     atomic.Int64 // monotonic JSON-RPC request ID counter
}

// session tracks MCP session state for a specific server URL.
type session struct {
	mu          sync.Mutex
	id          string // Mcp-Session-Id from server
	initialized bool
}

// --- JSON-RPC 2.0 wire types ---

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"` // nil for notifications
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// --- MCP-specific param/result types ---

type initializeParams struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    interface{}   `json:"capabilities"`
	ClientInfo      mcpClientInfo `json:"clientInfo"`
}

type mcpClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type toolsListResult struct {
	Tools []ToolInfo `json:"tools"`
}

type toolsCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError"`
}

type contentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
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

// Execute invokes a named tool on the MCP server using the tools/call method.
// The response content is extracted as text when possible.
func (c *HTTPClient) Execute(ctx context.Context, server ServerConfig, toolName string, params map[string]interface{}) (interface{}, error) {
	if err := c.validateURL(server.URL); err != nil {
		return nil, err
	}

	if err := c.ensureInitialized(ctx, server); err != nil {
		return nil, fmt.Errorf("mcp: initialize failed: %w", err)
	}

	result, err := c.call(ctx, server, "tools/call", toolsCallParams{
		Name:      toolName,
		Arguments: params,
	})
	if err != nil {
		return nil, err
	}

	var callResult toolsCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		// If not a standard tools/call result, return raw decoded JSON
		var raw interface{}
		if err2 := json.Unmarshal(result, &raw); err2 != nil {
			return nil, fmt.Errorf("mcp: failed to decode tool result: %w", err)
		}
		return raw, nil
	}

	if callResult.IsError {
		errText := extractTextContent(callResult.Content)
		return nil, fmt.Errorf("mcp: tool error: %s", errText)
	}

	text := extractTextContent(callResult.Content)
	if text != "" {
		return text, nil
	}

	// No text content — return raw decoded result for other content types
	var raw interface{}
	_ = json.Unmarshal(result, &raw)
	return raw, nil
}

// ListTools discovers tools from the MCP server using the tools/list method.
func (c *HTTPClient) ListTools(ctx context.Context, server ServerConfig) ([]ToolInfo, error) {
	if err := c.validateURL(server.URL); err != nil {
		return nil, err
	}

	if err := c.ensureInitialized(ctx, server); err != nil {
		return nil, fmt.Errorf("mcp: initialize failed: %w", err)
	}

	result, err := c.call(ctx, server, "tools/list", struct{}{})
	if err != nil {
		return nil, err
	}

	var listResult toolsListResult
	if err := json.Unmarshal(result, &listResult); err != nil {
		return nil, fmt.Errorf("mcp: failed to decode tools list: %w", err)
	}
	return listResult.Tools, nil
}

// ensureInitialized performs the MCP initialize handshake if not already done
// for the given server. Thread-safe per server URL.
func (c *HTTPClient) ensureInitialized(ctx context.Context, server ServerConfig) error {
	sess := c.getOrCreateSession(server.URL)
	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.initialized {
		return nil
	}

	clientName := c.opts.ClientName
	if clientName == "" {
		clientName = "agent-sdk-go"
	}
	clientVersion := c.opts.ClientVersion
	if clientVersion == "" {
		clientVersion = "1.0.0"
	}

	// Step 1: Send initialize request
	_, err := c.callWithSession(ctx, server, "initialize", initializeParams{
		ProtocolVersion: mcpProtocolVersion,
		Capabilities:    struct{}{},
		ClientInfo: mcpClientInfo{
			Name:    clientName,
			Version: clientVersion,
		},
	}, sess)
	if err != nil {
		return err
	}

	// Step 2: Send notifications/initialized notification
	if err := c.notifyWithSession(ctx, server, "notifications/initialized", sess); err != nil {
		return err
	}

	sess.initialized = true
	return nil
}

// call sends a JSON-RPC request and returns the result payload.
func (c *HTTPClient) call(ctx context.Context, server ServerConfig, method string, params interface{}) (json.RawMessage, error) {
	sess := c.getOrCreateSession(server.URL)
	return c.callWithSession(ctx, server, method, params, sess)
}

// callWithSession sends a JSON-RPC request using the provided session for headers.
func (c *HTTPClient) callWithSession(ctx context.Context, server ServerConfig, method string, params interface{}, sess *session) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	rpcReq := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to marshal request: %w", err)
	}

	ctx, cancel := c.withTimeout(ctx, server.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", mcpAcceptHeader)
	c.applyHeaders(httpReq, server)

	if sess.id != "" {
		httpReq.Header.Set("Mcp-Session-Id", sess.id)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("mcp: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Capture session ID from response
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		sess.id = sessionID
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited := io.LimitReader(resp.Body, c.opts.MaxResponseBytes)
		errBody, _ := io.ReadAll(limited)
		return nil, fmt.Errorf("mcp: server returned status %d: %s", resp.StatusCode, string(errBody))
	}

	// Parse response based on Content-Type
	respBody, err := c.readResponse(resp)
	if err != nil {
		return nil, err
	}

	var rpcResp jsonrpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("mcp: failed to decode JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// notifyWithSession sends a JSON-RPC notification (no id, no response expected).
func (c *HTTPClient) notifyWithSession(ctx context.Context, server ServerConfig, method string, sess *session) error {
	rpcReq := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return fmt.Errorf("mcp: failed to marshal notification: %w", err)
	}

	ctx, cancel := c.withTimeout(ctx, server.Timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mcp: failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", mcpAcceptHeader)
	c.applyHeaders(httpReq, server)

	if sess.id != "" {
		httpReq.Header.Set("Mcp-Session-Id", sess.id)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("mcp: notification failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	// Capture session ID from response
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		sess.id = sessionID
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp: notification returned status %d", resp.StatusCode)
	}

	return nil
}

// readResponse reads the HTTP response body, handling both JSON and SSE formats.
func (c *HTTPClient) readResponse(resp *http.Response) ([]byte, error) {
	contentType := resp.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "text/event-stream") {
		return c.parseSSE(resp.Body)
	}

	limited := io.LimitReader(resp.Body, c.opts.MaxResponseBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("mcp: failed to read response: %w", err)
	}
	return body, nil
}

// parseSSE extracts the last JSON-RPC message data line from an SSE stream.
// In the MCP protocol, SSE events use "event: message" with a "data:" line
// containing a complete JSON-RPC message.
func (c *HTTPClient) parseSSE(body io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(body)
	var lastData []byte

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			lastData = []byte(strings.TrimPrefix(line, "data: "))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("mcp: failed to read SSE stream: %w", err)
	}
	if lastData == nil {
		return nil, fmt.Errorf("mcp: empty SSE stream")
	}
	return lastData, nil
}

// extractTextContent joins all text content items from an MCP content array.
func extractTextContent(content []contentItem) string {
	var parts []string
	for _, item := range content {
		if item.Type == "text" && item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// getOrCreateSession returns or creates the session for a server URL.
func (c *HTTPClient) getOrCreateSession(serverURL string) *session {
	val, _ := c.sessions.LoadOrStore(serverURL, &session{})
	return val.(*session)
}

// withTimeout creates a context with the server-specific or default timeout.
func (c *HTTPClient) withTimeout(ctx context.Context, serverTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := serverTimeout
	if timeout == 0 {
		timeout = c.opts.DefaultTimeout
	}
	return context.WithTimeout(ctx, timeout)
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
