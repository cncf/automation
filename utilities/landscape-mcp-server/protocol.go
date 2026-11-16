package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
)

// JSON-RPC structures -------------------------------------------------------

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type jsonRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

var outputMu sync.Mutex

// isNotification reports whether the request carries no id and therefore must
// never receive a response.
func isNotification(req *jsonRPCRequest) bool {
	return len(req.ID) == 0
}

// nullID is the id of a response to a message whose own id is unknown.
var nullID = json.RawMessage("null")

// decodeRequest parses one line of input. A line that cannot be served yields
// the error response to send instead, so the client is never left waiting:
// -32700 for invalid JSON and -32600 for JSON that is not a single request
// object. Batch arrays are not supported and fall into the latter.
func decodeRequest(line []byte) (*jsonRPCRequest, *jsonRPCResponse) {
	if !json.Valid(line) {
		return nil, errorResponse(nullID, -32700, "Parse error", nil)
	}
	if bytes.HasPrefix(bytes.TrimSpace(line), []byte("[")) {
		return nil, errorResponse(nullID, -32600, "Invalid Request: batch requests are not supported", nil)
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		// Echo the id when it can still be read so the client can match the
		// error to its request.
		id := nullID
		var probe struct {
			ID json.RawMessage `json:"id"`
		}
		if json.Unmarshal(line, &probe) == nil && len(probe.ID) > 0 {
			id = probe.ID
		}
		return nil, errorResponse(id, -32600, "Invalid Request", nil)
	}
	return &req, nil
}

func handleRequest(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	resp := dispatchRequest(ctx, req, state)
	if isNotification(req) {
		return nil
	}
	return resp
}

func dispatchRequest(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"protocolVersion": negotiateProtocolVersion(req.Params),
				"capabilities": map[string]interface{}{
					"tools":     map[string]interface{}{},
					"resources": map[string]interface{}{},
					"prompts":   map[string]interface{}{},
				},
				"serverInfo": map[string]string{
					"name":    fmt.Sprintf("%s-landscape-mcp-server", strings.ToLower(state.cfg.Name)),
					"version": serverVersion,
				},
			}),
			ID: req.ID,
		}
	case "notifications/initialized":
		return nil
	case "ping":
		return &jsonRPCResponse{JSONRPC: "2.0", Result: json.RawMessage(`{}`), ID: req.ID}
	case "tools/list":
		return handleToolsList(req, state)
	case "tools/call":
		return handleToolsCall(ctx, req, state)
	case "resources/list":
		return handleResourcesList(req, state)
	case "resources/templates/list":
		return handleResourceTemplatesList(req, state)
	case "resources/read":
		return handleResourcesRead(ctx, req, state)
	case "prompts/list":
		return handlePromptsList(req, state)
	case "prompts/get":
		return handlePromptsGet(req, state)
	default:
		if isNotification(req) {
			log.Printf("ignoring unknown notification: %s", req.Method)
			return nil
		}
		return errorResponse(req.ID, -32601, "Method not found", nil)
	}
}

func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func writeResponse(resp *jsonRPCResponse) {
	outputMu.Lock()
	defer outputMu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("unable to marshal response: %v", err)
		return
	}
	fmt.Println(string(data))
}

func errorResponse(id json.RawMessage, code int, message string, data json.RawMessage) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}

// textContent is the MCP content array carrying a single text block.
func textContent(text string) []map[string]string {
	return []map[string]string{{"type": "text", "text": text}}
}

// textResult wraps text as a successful MCP tool result.
func textResult(id json.RawMessage, text string) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": textContent(text)}),
		ID:      id,
	}
}

// toolErrorResult reports a tool execution failure as a normal result flagged
// with isError, as MCP requires, rather than as a JSON-RPC protocol error.
func toolErrorResult(id json.RawMessage, message string) *jsonRPCResponse {
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result: mustJSON(map[string]interface{}{
			"content": textContent(message),
			"isError": true,
		}),
		ID: id,
	}
}

func datasetErrorText(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "Dataset unavailable: timed out waiting for dataset to load"
	}
	return "Dataset unavailable: " + err.Error()
}
