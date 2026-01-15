package mcp

import "encoding/json"

// JSON-RPC version constant.
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NewRequest creates a new JSON-RPC request.
func NewRequest(id int64, method string, params any) (*Request, error) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsJSON = data
	}
	return &Request{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	return e.Message
}

// Standard JSON-RPC error codes.
const (
	ErrorCodeParseError     = -32700
	ErrorCodeInvalidRequest = -32600
	ErrorCodeMethodNotFound = -32601
	ErrorCodeInvalidParams  = -32602
	ErrorCodeInternalError  = -32603
)

// MCP-specific method names.
const (
	MethodInitialize = "initialize"
	MethodToolsList  = "tools/list"
	MethodToolsCall  = "tools/call"
	MethodShutdown   = "shutdown"
)

// InitializeParams are the parameters for the initialize method.
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// ClientCapabilities describes what the client supports.
type ClientCapabilities struct {
	// Currently empty - can be extended as needed
}

// ClientInfo identifies the client to the server.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult is the result of the initialize method.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      *ServerInfoResult  `json:"serverInfo,omitempty"`
}

// ServerInfoResult contains server identification info.
type ServerInfoResult struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ToolsListResult is the result of the tools/list method.
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// ToolCallParams are the parameters for the tools/call method.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult is the result of the tools/call method.
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a block of content in a tool result.
type ContentBlock struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// TextContent returns the concatenated text content from all text blocks.
func (r *ToolCallResult) TextContent() string {
	var text string
	for _, block := range r.Content {
		if block.Type == "text" {
			text += block.Text
		}
	}
	return text
}
