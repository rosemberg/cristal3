// Package mcp implements the Model Context Protocol (MCP) over stdio transport.
// Protocol revision: 2025-11-25.
package mcp

import "encoding/json"

// ProtocolVersion is the MCP protocol revision this server implements.
const ProtocolVersion = "2025-11-25"

// JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700 // invalid JSON
	CodeInvalidRequest = -32600 // not a valid Request object
	CodeMethodNotFound = -32601 // method not found
	CodeInvalidParams  = -32602 // invalid method parameters
	CodeInternalError  = -32603 // internal JSON-RPC error
)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 envelope types
// ---------------------------------------------------------------------------

// Request is a JSON-RPC 2.0 request object.
// ID is json.RawMessage so it can hold a string, number, or null.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response object.
// Exactly one of Result or Error MUST be set per the spec.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification is a JSON-RPC 2.0 notification (no ID, no response expected).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError is the error object inside a Response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ---------------------------------------------------------------------------
// MCP initialize
// ---------------------------------------------------------------------------

// InitializeParams is the params object for the "initialize" request.
type InitializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ClientInfo      ClientInfo `json:"clientInfo"`
	Capabilities    any        `json:"capabilities,omitempty"`
}

// InitializeResult is the result of the "initialize" request.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
	Capabilities    ServerCapabilities `json:"capabilities"`
}

// ClientInfo identifies the connecting MCP client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerInfo identifies this server in the initialize response.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities declares which MCP feature sets this server supports.
// Only tools is advertised; resources/prompts/sampling/roots are omitted.
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability is the nested object under capabilities.tools.
// An empty struct serialises to {} which is the correct declaration.
type ToolsCapability struct{}

// ---------------------------------------------------------------------------
// MCP tools/list
// ---------------------------------------------------------------------------

// Tool describes a single tool in the tools/list response.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult is the result of "tools/list".
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

// ---------------------------------------------------------------------------
// MCP tools/call
// ---------------------------------------------------------------------------

// CallToolParams is the params object for "tools/call".
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ---------------------------------------------------------------------------
// MCP notifications/cancelled
// ---------------------------------------------------------------------------

// CancelledNotification is the params of "notifications/cancelled".
type CancelledNotification struct {
	RequestID json.RawMessage `json:"requestId"`
	Reason    string          `json:"reason,omitempty"`
}
