package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"piper/internal/engine"
	"piper/internal/types"
)

// MCPServer implements a JSON-RPC based MCP (Model Context Protocol) server
// that exposes flows as tools. It reads from stdin and writes to stdout.
type MCPServer struct {
	engine *engine.Engine
	flows  map[string]*types.FlowDef
}

// NewMCPServer creates a new MCP server.
func NewMCPServer(eng *engine.Engine, flows map[string]*types.FlowDef) *MCPServer {
	return &MCPServer{engine: eng, flows: flows}
}

// JSON-RPC types
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol types
type mcpInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      mcpServerInfo  `json:"serverInfo"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpToolsResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpCallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpCallToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ServeStdio runs the MCP server on stdin/stdout.
func (s *MCPServer) ServeStdio() error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req jsonRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decoding request: %w", err)
		}

		resp := s.handleRequest(req)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return fmt.Errorf("encoding response: %w", err)
			}
		}
	}
}

func (s *MCPServer) handleRequest(req jsonRPCRequest) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcpInitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: map[string]any{
					"tools": map[string]any{},
				},
				ServerInfo: mcpServerInfo{
					Name:    "piper",
					Version: "0.1.0",
				},
			},
		}

	case "notifications/initialized":
		// No response needed for notifications.
		return nil

	case "tools/list":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  s.listTools(),
		}

	case "tools/call":
		var params mcpCallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   jsonRPCError{Code: -32602, Message: "invalid params: " + err.Error()},
			}
		}
		result, isError := s.callTool(params)
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcpCallToolResult{
				Content: []mcpContent{{Type: "text", Text: result}},
				IsError: isError,
			},
		}

	default:
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   jsonRPCError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}

func (s *MCPServer) listTools() mcpToolsResult {
	tools := make([]mcpTool, 0, len(s.flows))
	for _, flow := range s.flows {
		tool := mcpTool{
			Name:        flow.Name,
			Description: flow.Description,
			InputSchema: s.buildInputSchema(flow),
		}
		tools = append(tools, tool)
	}
	return mcpToolsResult{Tools: tools}
}

func (s *MCPServer) buildInputSchema(flow *types.FlowDef) map[string]any {
	schema := map[string]any{
		"type": "object",
	}

	if flow.Input == nil || len(flow.Input.Properties) == 0 {
		return schema
	}

	properties := make(map[string]any)
	var required []string

	for name, field := range flow.Input.Properties {
		prop := map[string]any{
			"type": field.Type,
		}
		if field.Description != "" {
			prop["description"] = field.Description
		}
		properties[name] = prop
		if field.Required {
			required = append(required, name)
		}
	}

	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func (s *MCPServer) callTool(params mcpCallToolParams) (string, bool) {
	flow, ok := s.flows[params.Name]
	if !ok {
		return fmt.Sprintf("flow %q not found", params.Name), true
	}

	result, err := s.engine.Run(context.Background(), flow, params.Arguments)
	if err != nil {
		return fmt.Sprintf("error: %v", err), true
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Sprintf("error marshaling result: %v", err), true
	}

	return string(resultJSON), result.Status == "failed"
}
