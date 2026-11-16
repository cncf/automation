package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// MCP Resources handlers ----------------------------------------------------

func handleResourcesList(req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	resources := []map[string]string{
		{
			"uri":         "landscape://categories",
			"name":        fmt.Sprintf("%s Categories", state.cfg.Name),
			"description": "Complete category and subcategory tree with item counts",
			"mimeType":    "application/json",
		},
		{
			"uri":         "landscape://summary",
			"name":        fmt.Sprintf("%s Summary", state.cfg.Name),
			"description": "High-level landscape statistics and overview",
			"mimeType":    "application/json",
		},
	}
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"resources": resources}),
		ID:      req.ID,
	}
}

// errCodeResourceNotFound is the MCP error code for a resource that does not
// exist.
const errCodeResourceNotFound = -32002

// resourceErrorResponse maps a resources/read failure to its JSON-RPC error: a
// URI that names nothing is -32002, any other fault of the caller's URI is
// -32602, and everything else is an internal error.
func resourceErrorResponse(id json.RawMessage, uri string, err error, callerFault bool) *jsonRPCResponse {
	var notFound *notFoundError
	switch {
	case errors.As(err, &notFound):
		return errorResponse(id, errCodeResourceNotFound, err.Error(), mustJSON(map[string]string{"uri": uri}))
	case callerFault:
		return errorResponse(id, -32602, err.Error(), nil)
	default:
		return errorResponse(id, -32000, err.Error(), nil)
	}
}

func handleResourcesRead(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", nil)
	}

	if params.URI == "" {
		return errorResponse(req.ID, -32602, "uri parameter is required", nil)
	}

	reader, err := resolveResource(params.URI)
	if err != nil {
		return resourceErrorResponse(req.ID, params.URI, err, true)
	}

	waitCtx, cancel := context.WithTimeout(ctx, datasetWaitTimeout)
	defer cancel()
	dataset, err := state.waitForDataset(waitCtx)
	if err != nil {
		return errorResponse(req.ID, -32603, datasetErrorText(err), mustJSON(map[string]string{"error": err.Error()}))
	}

	text, err := reader.read(dataset)
	if err != nil {
		return resourceErrorResponse(req.ID, params.URI, err, reader.byName)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result: mustJSON(map[string]interface{}{
			"contents": []map[string]string{
				{
					"uri":      params.URI,
					"mimeType": "application/json",
					"text":     text,
				},
			},
		}),
		ID: req.ID,
	}
}
