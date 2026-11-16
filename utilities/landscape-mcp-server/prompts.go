package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// MCP Prompts handlers ------------------------------------------------------

func handlePromptsList(req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	prompts := []map[string]interface{}{
		{
			"name":        "analyze_landscape",
			"description": fmt.Sprintf("Analyze the current state of the %s landscape", state.cfg.Name),
			"arguments":   []interface{}{},
		},
		{
			"name":        "compare_projects",
			"description": fmt.Sprintf("Compare specific projects in the %s landscape", state.cfg.Name),
			"arguments": []map[string]interface{}{
				{
					"name":        "project_names",
					"description": "Comma-separated list of project names to compare",
					"required":    true,
				},
			},
		},
		{
			"name":        "landscape_changes_report",
			"description": fmt.Sprintf("Report promotions, new sandbox projects, and archived projects in the %s landscape", state.cfg.Name),
			"arguments": []map[string]interface{}{
				{
					"name":        "since",
					"description": "Report changes since this date; defaults to 12 months ago",
					"required":    false,
				},
			},
		},
	}
	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"prompts": prompts}),
		ID:      req.ID,
	}
}

func handlePromptsGet(req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	var params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", nil)
	}

	switch params.Name {
	case "analyze_landscape":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"description": fmt.Sprintf("Analyze the current state of the %s landscape", state.cfg.Name),
				"messages": []map[string]interface{}{
					{
						"role": "user",
						"content": map[string]string{
							"type": "text",
							"text": fmt.Sprintf(`Please analyze the current state of the %s landscape. Use the available tools to:
1. Get a landscape summary using the landscape_summary tool
2. List all categories using the list_categories tool
3. Identify trends in project maturity levels, using the query_projects tool to list projects by maturity
4. Analyze membership distribution across tiers
5. Provide insights and recommendations based on the data`, state.cfg.Name),
						},
					},
				},
			}),
			ID: req.ID,
		}
	case "compare_projects":
		projectNames := params.Arguments["project_names"]
		if projectNames == "" {
			return errorResponse(req.ID, -32602, "Missing required argument: project_names", nil)
		}
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"description": fmt.Sprintf("Compare specific projects in the %s landscape", state.cfg.Name),
				"messages": []map[string]interface{}{
					{
						"role": "user",
						"content": map[string]string{
							"type": "text",
							"text": fmt.Sprintf(`Please compare the following %s projects: %s

Use the compare_projects tool to get detailed information, then analyze:
1. Maturity levels and timeline to graduation
2. Repository activity and community health
3. Organizational backing and funding
4. Key similarities and differences
5. Recommendations for each project's growth path`, state.cfg.Name, projectNames),
						},
					},
				},
			}),
			ID: req.ID,
		}
	case "landscape_changes_report":
		text, err := changesReportPrompt(state.cfg.Name, params.Arguments["since"], time.Now())
		if err != nil {
			return errorResponse(req.ID, -32602, err.Error(), nil)
		}
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"description": fmt.Sprintf("Report promotions, new sandbox projects, and archived projects in the %s landscape", state.cfg.Name),
				"messages": []map[string]interface{}{
					{
						"role":    "user",
						"content": map[string]string{"type": "text", "text": text},
					},
				},
			}),
			ID: req.ID,
		}
	default:
		return errorResponse(req.ID, -32602, fmt.Sprintf("Unknown prompt: %s", params.Name), nil)
	}
}
