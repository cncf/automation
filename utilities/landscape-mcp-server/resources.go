package main

import (
	"context"
	"encoding/json"
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

func handleResourcesRead(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", nil)
	}

	reader, err := resolveResource(params.URI)
	if err != nil {
		return errorResponse(req.ID, -32602, err.Error(), nil)
	}

	waitCtx, cancel := context.WithTimeout(ctx, datasetWaitTimeout)
	defer cancel()
	dataset, err := state.waitForDataset(waitCtx)
	if err != nil {
		return errorResponse(req.ID, -32603, datasetErrorText(err), mustJSON(map[string]string{"error": err.Error()}))
	}

	text, err := reader.read(dataset)
	if err != nil {
		code := -32000
		if reader.notFoundIsInvalidParams {
			code = -32602
		}
		return errorResponse(req.ID, code, err.Error(), nil)
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
