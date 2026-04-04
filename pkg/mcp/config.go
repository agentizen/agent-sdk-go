package mcp

import "time"

// ServerConfig describes an MCP server and its transport.
// Each server carries its own Client so an agent can mix HTTP, stdio, etc.
type ServerConfig struct {
	Handle      string            // unique identifier ("github", "stripe")
	URL         string            // base URL (for HTTP) or command (for stdio)
	Description string            // human-readable description
	Headers     map[string]string // static headers added to every request
	Timeout     time.Duration     // per-request timeout (default 30s)
	Client      Client            // transport to use for this server
}

// ClientOptions configures the HTTP MCP client transport.
type ClientOptions struct {
	AllowHTTP        bool          // allow http:// URLs (dev only, default false)
	MaxResponseBytes int64         // max response body size (default 10MB)
	DefaultTimeout   time.Duration // default per-request timeout (default 30s)
	ClientName       string        // client name sent during initialize (default "agent-sdk-go")
	ClientVersion    string        // client version sent during initialize (default "1.0.0")
}
