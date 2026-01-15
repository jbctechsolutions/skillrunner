package mcp

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name   string
		id     int64
		method string
		params any
		want   *Request
	}{
		{
			name:   "request without params",
			id:     1,
			method: MethodInitialize,
			params: nil,
			want: &Request{
				JSONRPC: JSONRPCVersion,
				ID:      1,
				Method:  MethodInitialize,
				Params:  nil,
			},
		},
		{
			name:   "request with params",
			id:     2,
			method: MethodToolsCall,
			params: ToolCallParams{Name: "test_tool"},
			want: &Request{
				JSONRPC: JSONRPCVersion,
				ID:      2,
				Method:  MethodToolsCall,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRequest(tt.id, tt.method, tt.params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.JSONRPC != tt.want.JSONRPC {
				t.Errorf("JSONRPC = %q, want %q", got.JSONRPC, tt.want.JSONRPC)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.want.ID)
			}
			if got.Method != tt.want.Method {
				t.Errorf("Method = %q, want %q", got.Method, tt.want.Method)
			}
			if tt.params == nil && got.Params != nil {
				t.Error("Params should be nil")
			}
			if tt.params != nil && got.Params == nil {
				t.Error("Params should not be nil")
			}
		})
	}
}

func TestNewRequest_InvalidParams(t *testing.T) {
	// Create an unmarshallable value (channel)
	ch := make(chan int)
	_, err := NewRequest(1, "test", ch)
	if err == nil {
		t.Error("expected error for unmarshallable params")
	}
}

func TestRequest_JSON(t *testing.T) {
	req, err := NewRequest(1, MethodInitialize, InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: ClientInfo{
			Name:    "skillrunner",
			Version: "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	if decoded.JSONRPC != req.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, req.JSONRPC)
	}
	if decoded.ID != req.ID {
		t.Errorf("ID = %d, want %d", decoded.ID, req.ID)
	}
	if decoded.Method != req.Method {
		t.Errorf("Method = %q, want %q", decoded.Method, req.Method)
	}
}

func TestResponse_JSON(t *testing.T) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      1,
		Result:  json.RawMessage(`{"protocolVersion":"2024-11-05"}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.JSONRPC != resp.JSONRPC {
		t.Errorf("JSONRPC = %q, want %q", decoded.JSONRPC, resp.JSONRPC)
	}
	if decoded.ID != resp.ID {
		t.Errorf("ID = %d, want %d", decoded.ID, resp.ID)
	}
	if decoded.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestResponse_WithError(t *testing.T) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      1,
		Error: &RPCError{
			Code:    ErrorCodeMethodNotFound,
			Message: "method not found",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if decoded.Error.Code != ErrorCodeMethodNotFound {
		t.Errorf("Error.Code = %d, want %d", decoded.Error.Code, ErrorCodeMethodNotFound)
	}
	if decoded.Error.Message != "method not found" {
		t.Errorf("Error.Message = %q, want %q", decoded.Error.Message, "method not found")
	}
}

func TestRPCError_Error(t *testing.T) {
	err := &RPCError{
		Code:    ErrorCodeInvalidParams,
		Message: "invalid parameters",
	}

	if got := err.Error(); got != "invalid parameters" {
		t.Errorf("Error() = %q, want %q", got, "invalid parameters")
	}
}

func TestInitializeParams_JSON(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    ClientCapabilities{},
		ClientInfo: ClientInfo{
			Name:    "skillrunner",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	var decoded InitializeParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	if decoded.ProtocolVersion != params.ProtocolVersion {
		t.Errorf("ProtocolVersion = %q, want %q", decoded.ProtocolVersion, params.ProtocolVersion)
	}
	if decoded.ClientInfo.Name != params.ClientInfo.Name {
		t.Errorf("ClientInfo.Name = %q, want %q", decoded.ClientInfo.Name, params.ClientInfo.Name)
	}
}

func TestInitializeResult_JSON(t *testing.T) {
	jsonStr := `{
		"protocolVersion": "2024-11-05",
		"capabilities": {
			"tools": {"listChanged": true}
		},
		"serverInfo": {
			"name": "linear-mcp-server",
			"version": "1.0.0"
		}
	}`

	var result InitializeResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", result.ProtocolVersion, "2024-11-05")
	}
	if result.ServerInfo == nil {
		t.Fatal("ServerInfo should not be nil")
	}
	if result.ServerInfo.Name != "linear-mcp-server" {
		t.Errorf("ServerInfo.Name = %q, want %q", result.ServerInfo.Name, "linear-mcp-server")
	}
	if result.Capabilities.Tools == nil {
		t.Error("Capabilities.Tools should not be nil")
	}
	if !result.Capabilities.Tools.ListChanged {
		t.Error("Capabilities.Tools.ListChanged should be true")
	}
}

func TestToolsListResult_JSON(t *testing.T) {
	jsonStr := `{
		"tools": [
			{
				"name": "create_issue",
				"description": "Creates a new issue",
				"inputSchema": {"type": "object", "properties": {"title": {"type": "string"}}}
			},
			{
				"name": "list_issues",
				"description": "Lists all issues"
			}
		]
	}`

	var result ToolsListResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(result.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want %d", len(result.Tools), 2)
	}

	if result.Tools[0].Name != "create_issue" {
		t.Errorf("Tools[0].Name = %q, want %q", result.Tools[0].Name, "create_issue")
	}
	if result.Tools[1].Name != "list_issues" {
		t.Errorf("Tools[1].Name = %q, want %q", result.Tools[1].Name, "list_issues")
	}
}

func TestToolCallParams_JSON(t *testing.T) {
	params := ToolCallParams{
		Name:      "create_issue",
		Arguments: json.RawMessage(`{"title": "Test issue"}`),
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	var decoded ToolCallParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	if decoded.Name != params.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, params.Name)
	}
}

func TestToolCallResult_JSON(t *testing.T) {
	jsonStr := `{
		"content": [
			{"type": "text", "text": "Issue created successfully"},
			{"type": "text", "text": " with ID 123"}
		],
		"isError": false
	}`

	var result ToolCallResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(result.Content) != 2 {
		t.Fatalf("len(Content) = %d, want %d", len(result.Content), 2)
	}
	if result.IsError {
		t.Error("IsError should be false")
	}
}

func TestToolCallResult_TextContent(t *testing.T) {
	tests := []struct {
		name   string
		result ToolCallResult
		want   string
	}{
		{
			name: "single text block",
			result: ToolCallResult{
				Content: []ContentBlock{
					{Type: "text", Text: "Hello"},
				},
			},
			want: "Hello",
		},
		{
			name: "multiple text blocks",
			result: ToolCallResult{
				Content: []ContentBlock{
					{Type: "text", Text: "Hello"},
					{Type: "text", Text: " World"},
				},
			},
			want: "Hello World",
		},
		{
			name: "mixed content types",
			result: ToolCallResult{
				Content: []ContentBlock{
					{Type: "text", Text: "Start"},
					{Type: "image", Text: ""},
					{Type: "text", Text: "End"},
				},
			},
			want: "StartEnd",
		},
		{
			name: "empty content",
			result: ToolCallResult{
				Content: []ContentBlock{},
			},
			want: "",
		},
		{
			name: "no text blocks",
			result: ToolCallResult{
				Content: []ContentBlock{
					{Type: "image"},
					{Type: "resource"},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.TextContent(); got != tt.want {
				t.Errorf("TextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMethodConstants(t *testing.T) {
	// Ensure method constants match MCP spec
	if MethodInitialize != "initialize" {
		t.Errorf("MethodInitialize = %q, want %q", MethodInitialize, "initialize")
	}
	if MethodToolsList != "tools/list" {
		t.Errorf("MethodToolsList = %q, want %q", MethodToolsList, "tools/list")
	}
	if MethodToolsCall != "tools/call" {
		t.Errorf("MethodToolsCall = %q, want %q", MethodToolsCall, "tools/call")
	}
	if MethodShutdown != "shutdown" {
		t.Errorf("MethodShutdown = %q, want %q", MethodShutdown, "shutdown")
	}
}

func TestErrorCodeConstants(t *testing.T) {
	// Ensure error codes match JSON-RPC spec
	if ErrorCodeParseError != -32700 {
		t.Errorf("ErrorCodeParseError = %d, want %d", ErrorCodeParseError, -32700)
	}
	if ErrorCodeInvalidRequest != -32600 {
		t.Errorf("ErrorCodeInvalidRequest = %d, want %d", ErrorCodeInvalidRequest, -32600)
	}
	if ErrorCodeMethodNotFound != -32601 {
		t.Errorf("ErrorCodeMethodNotFound = %d, want %d", ErrorCodeMethodNotFound, -32601)
	}
	if ErrorCodeInvalidParams != -32602 {
		t.Errorf("ErrorCodeInvalidParams = %d, want %d", ErrorCodeInvalidParams, -32602)
	}
	if ErrorCodeInternalError != -32603 {
		t.Errorf("ErrorCodeInternalError = %d, want %d", ErrorCodeInternalError, -32603)
	}
}

func TestJSONRPCVersion(t *testing.T) {
	if JSONRPCVersion != "2.0" {
		t.Errorf("JSONRPCVersion = %q, want %q", JSONRPCVersion, "2.0")
	}
}
