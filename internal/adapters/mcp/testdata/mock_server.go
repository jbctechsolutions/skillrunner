//go:build ignore

// This is a mock MCP server for testing purposes.
// It responds to JSON-RPC requests over stdio.
package main

import (
	"bufio"
	"encoding/json"
	"os"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		resp := Response{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "initialize":
			resp.Result = map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]interface{}{
					"name":    "mock-mcp-server",
					"version": "1.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
				},
			}

		case "tools/list":
			resp.Result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "test_tool",
						"description": "A test tool",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"message": map[string]interface{}{
									"type":        "string",
									"description": "A test message",
								},
							},
						},
					},
					{
						"name":        "echo",
						"description": "Echoes the input",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"text": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			}

		case "tools/call":
			var params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)

			switch params.Name {
			case "test_tool":
				resp.Result = map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": "Test tool executed successfully",
						},
					},
				}
			case "echo":
				var args struct {
					Text string `json:"text"`
				}
				json.Unmarshal(params.Arguments, &args)
				resp.Result = map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": args.Text,
						},
					},
				}
			default:
				resp.Error = &RPCError{
					Code:    -32601,
					Message: "tool not found: " + params.Name,
				}
			}

		case "shutdown":
			resp.Result = map[string]interface{}{}
			encoder.Encode(resp)
			os.Exit(0)

		default:
			resp.Error = &RPCError{
				Code:    -32601,
				Message: "method not found: " + req.Method,
			}
		}

		encoder.Encode(resp)
	}
}
