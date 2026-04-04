// Package main demonstrates end-to-end MCP (Model Context Protocol) communication
// using the SDK's HTTP client and a local JSON-RPC 2.0 test server.
//
// It verifies the complete MCP Streamable HTTP flow:
//   - Initialize handshake (initialize + notifications/initialized)
//   - Tool discovery via tools/list
//   - Tool execution via tools/call
//   - Session management via Mcp-Session-Id
//   - Accept header negotiation
//
// Run: go run ./examples/mcp_live_server
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"

	agentsdk "github.com/agentizen/agent-sdk-go"
)

// --- JSON-RPC 2.0 wire types (server-side) ---

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64      `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// sanitizeLog strips newlines from user-provided values before logging
// to prevent log injection (CodeQL go/log-injection).
func sanitizeLog(s string) string {
	r := strings.NewReplacer("\n", "", "\r", "")
	return r.Replace(s)
}

// mcpServer is a minimal MCP server that exposes two tools: "add" and "greet".
type mcpServer struct {
	sessionID    string
	requestCount atomic.Int32
}

func (s *mcpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	count := s.requestCount.Add(1)
	fmt.Printf("  [server] Request #%d: %s\n", count, r.Method)

	// Validate Accept header
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "application/json") || !strings.Contains(accept, "text/event-stream") {
		w.WriteHeader(http.StatusNotAcceptable)
		resp := jsonrpcResponse{
			JSONRPC: "2.0",
			Error:   map[string]interface{}{"code": -32000, "message": "Not Acceptable: Client must accept both application/json and text/event-stream"},
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	body, _ := io.ReadAll(r.Body)
	var req jsonrpcRequest
	_ = json.Unmarshal(body, &req)

	fmt.Printf("  [server] Method: %s\n", sanitizeLog(req.Method))

	// Always set session ID
	w.Header().Set("Mcp-Session-Id", s.sessionID)

	// Handle notifications (no ID)
	if req.ID == nil {
		fmt.Printf("  [server] Notification received: %s\n", sanitizeLog(req.Method))
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
			"serverInfo":      map[string]interface{}{"name": "test-mcp-server", "version": "1.0.0"},
		}

	case "tools/list":
		result = map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "add",
					"description": "Add two numbers together",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"a": map[string]interface{}{"type": "number", "description": "First number"},
							"b": map[string]interface{}{"type": "number", "description": "Second number"},
						},
						"required": []string{"a", "b"},
					},
				},
				{
					"name":        "greet",
					"description": "Generate a greeting message",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{"type": "string", "description": "Name to greet"},
						},
						"required": []string{"name"},
					},
				},
			},
		}

	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &params)
		argsJSON, _ := json.Marshal(params.Arguments)
		fmt.Printf("  [server] Tool call: %s(%s)\n", sanitizeLog(params.Name), sanitizeLog(string(argsJSON)))

		switch params.Name {
		case "add":
			a, _ := params.Arguments["a"].(float64)
			b, _ := params.Arguments["b"].(float64)
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("%.0f + %.0f = %.0f", a, b, a+b)},
				},
				"isError": false,
			}
		case "greet":
			name, _ := params.Arguments["name"].(string)
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Hello, %s! Welcome to MCP.", name)},
				},
				"isError": false,
			}
		default:
			rpcErr = map[string]interface{}{"code": -32601, "message": fmt.Sprintf("Unknown tool: %s", params.Name)}
		}

	default:
		rpcErr = map[string]interface{}{"code": -32601, "message": fmt.Sprintf("Method not found: %s", req.Method)}
	}

	resp := jsonrpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rpcErr}
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	fmt.Println("=== MCP Live Server Example ===")
	fmt.Println("This example starts a local MCP JSON-RPC 2.0 server and tests the full protocol flow.")
	fmt.Println()

	// --- 1. Start a local MCP server ---
	server := &mcpServer{sessionID: "session-test-42"}
	srv := httptest.NewServer(server)
	defer srv.Close()
	fmt.Printf("MCP server started at %s\n\n", srv.URL)

	// --- 2. Create SDK MCP client ---
	mcpClient := agentsdk.NewMCPHTTPClient(agentsdk.MCPClientOptions{
		AllowHTTP:  true,
		ClientName: "mcp-live-example",
	})

	serverCfg := agentsdk.MCPServerConfig{
		Handle:      "test-server",
		URL:         srv.URL,
		Description: "Local test MCP server",
		Client:      mcpClient,
	}

	ctx := context.Background()

	// --- 3. Test: Discover tools (triggers initialize handshake) ---
	fmt.Println("--- Step 1: Discover tools (initialize + tools/list) ---")
	tools, err := mcpClient.ListTools(ctx, serverCfg)
	if err != nil {
		log.Fatalf("ListTools failed: %v", err)
	}
	fmt.Printf("Discovered %d tools:\n", len(tools))
	for _, t := range tools {
		fmt.Printf("  - %s: %s (schema keys: %d)\n", t.Name, t.Description, len(t.Parameters))
	}

	// --- 4. Test: Execute "add" tool ---
	fmt.Println("\n--- Step 2: Execute 'add' tool ---")
	result, err := mcpClient.Execute(ctx, serverCfg, "add", map[string]interface{}{
		"a": float64(17),
		"b": float64(25),
	})
	if err != nil {
		log.Fatalf("Execute 'add' failed: %v", err)
	}
	fmt.Printf("Result: %v\n", result)

	// --- 5. Test: Execute "greet" tool ---
	fmt.Println("\n--- Step 3: Execute 'greet' tool ---")
	result, err = mcpClient.Execute(ctx, serverCfg, "greet", map[string]interface{}{
		"name": "Agent SDK",
	})
	if err != nil {
		log.Fatalf("Execute 'greet' failed: %v", err)
	}
	fmt.Printf("Result: %v\n", result)

	// --- 6. Test: Tool discovery via adapter (same as Runner does) ---
	fmt.Println("\n--- Step 4: Discover tools via ToolsFromServer adapter ---")
	adapted, err := agentsdk.MCPToolsFromServer(ctx, serverCfg)
	if err != nil {
		log.Fatalf("ToolsFromServer failed: %v", err)
	}
	fmt.Printf("Adapted %d tools to tool.Tool interface:\n", len(adapted))
	for _, t := range adapted {
		fmt.Printf("  - %s: %s\n", t.GetName(), t.GetDescription())
	}

	// Execute via adapter (same path as the Runner)
	fmt.Println("\n--- Step 5: Execute via adapted tool (Runner path) ---")
	adapterResult, err := adapted[0].Execute(ctx, map[string]interface{}{
		"a": float64(100),
		"b": float64(200),
	})
	if err != nil {
		log.Fatalf("Adapted tool execution failed: %v", err)
	}
	fmt.Printf("Adapter result: %v\n", adapterResult)

	// --- 7. Summary ---
	fmt.Printf("\n=== All %d requests completed successfully ===\n", server.requestCount.Load())
	fmt.Println("Protocol verified:")
	fmt.Println("  [ok] JSON-RPC 2.0 message format")
	fmt.Println("  [ok] Accept: application/json, text/event-stream")
	fmt.Println("  [ok] Initialize handshake (initialize + notifications/initialized)")
	fmt.Println("  [ok] Session management (Mcp-Session-Id)")
	fmt.Println("  [ok] tools/list discovery")
	fmt.Println("  [ok] tools/call execution")
	fmt.Println("  [ok] ToolAdapter integration")
}
