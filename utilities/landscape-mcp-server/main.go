package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
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

// Server state --------------------------------------------------------------

type serverState struct {
	once    sync.Once
	ready   chan struct{}
	mu      sync.RWMutex
	dataset *Dataset
	loadErr error
}

var outputMu sync.Mutex

type toolDefinition struct {
	Name        string
	Description string
	Metrics     []string
}

var (
	toolCatalog = map[string]toolDefinition{
		"project_metrics": {
			Name:        "project_metrics",
			Description: "Answer questions about CNCF project counts and progress.",
			Metrics: []string{
				"incubating_project_count",
				"sandbox_projects_joined_this_year",
				"projects_graduated_last_year",
			},
		},
		"membership_metrics": {
			Name:        "membership_metrics",
			Description: "Answer questions about CNCF membership activity and funding.",
			Metrics: []string{
				"gold_members_joined_this_year",
				"silver_members_joined_this_year",
				"silver_members_raised_last_month",
				"gold_members_raised_this_year",
			},
		},
		"query_projects": {
			Name:        "query_projects",
			Description: "Query CNCF projects with flexible filtering by maturity, dates, and name. Returns detailed project information.",
		},
		"query_members": {
			Name:        "query_members",
			Description: "Query CNCF members with flexible filtering by membership tier and join dates.",
		},
		"get_project_details": {
			Name:        "get_project_details",
			Description: "Get detailed information about a specific CNCF project by name.",
		},
		// Legacy aggregated tool retained for compatibility with earlier clients.
		"query_landscape": {
			Name:        "query_landscape",
			Description: "Run predefined analytical queries over the CNCF landscape dataset.",
			Metrics: []string{
				"incubating_project_count",
				"sandbox_projects_joined_this_year",
				"projects_graduated_last_year",
				"gold_members_joined_this_year",
				"silver_members_joined_this_year",
				"silver_members_raised_last_month",
				"gold_members_raised_this_year",
			},
		},
	}
	advertisedTools = []string{
		"query_projects",
		"query_members",
		"get_project_details",
		"project_metrics",
		"membership_metrics",
	}
)

func newServerState() *serverState {
	return &serverState{ready: make(chan struct{})}
}

func (s *serverState) setDataset(ds *Dataset, err error) {
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.dataset = ds
		s.loadErr = err
		close(s.ready)
	})
}

func (s *serverState) waitForDataset(ctx context.Context) (*Dataset, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.ready:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.loadErr != nil {
		return nil, s.loadErr
	}
	if s.dataset == nil {
		return nil, errors.New("dataset not loaded")
	}
	return s.dataset, nil
}

// Main ----------------------------------------------------------------------

func main() {
	dataFile := flag.String("data-file", "", "Path to the landscape full dataset JSON file")
	dataURL := flag.String("data-url", "https://landscape.cncf.io/data/full.json", "URL to the landscape full dataset JSON file")
	flag.Parse()

	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	state := newServerState()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ds, err := loadDataset(ctx, dataSource{
			filePath: *dataFile,
			url:      *dataURL,
		})
		state.setDataset(ds, err)
		if err != nil {
			log.Printf("error loading dataset: %v", err)
			sendNotification("notifications/serverReady", map[string]interface{}{"error": err.Error()})
		} else {
			log.Printf("dataset loaded (%d items)", len(ds.Items))
			sendNotification("notifications/serverReady", map[string]interface{}{})
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("unable to parse JSON-RPC request: %v", err)
			continue
		}

		if resp := handleRequest(ctx, &req, state); resp != nil {
			writeResponse(resp)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdin scanner error: %v", err)
	}
}

func handleRequest(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	switch req.Method {
	case "initialize":
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]string{
					"name":    "landscape2-mcp-server-go",
					"version": "0.1.0",
				},
			}),
			ID: req.ID,
		}
	case "notifications/initialized":
		return nil
	case "tools/list":
		tools := make([]map[string]interface{}, 0, len(advertisedTools))
		for _, name := range advertisedTools {
			def, ok := toolCatalog[name]
			if !ok {
				continue
			}
			tools = append(tools, map[string]interface{}{
				"name":        def.Name,
				"description": def.Description,
				"inputSchema": toolInputSchema(def),
			})
		}
		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result: mustJSON(map[string]interface{}{
				"tools": tools,
			}),
			ID: req.ID,
		}
	case "tools/call":
		return handleToolsCall(ctx, req, state)
	default:
		return errorResponse(req.ID, -32601, "Method not found", nil)
	}
}

func handleToolsCall(ctx context.Context, req *jsonRPCRequest, state *serverState) *jsonRPCResponse {
	var payload struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", nil)
	}

	def, ok := toolCatalog[payload.Name]
	if !ok {
		return errorResponse(req.ID, -32601, "Tool not found", nil)
	}

	dataset, err := state.waitForDataset(ctx)
	if err != nil {
		return errorResponse(req.ID, -32603, "Dataset unavailable", mustJSON(map[string]string{"error": err.Error()}))
	}

	// Handle different tool types
	switch payload.Name {
	case "query_projects":
		return handleQueryProjects(req.ID, payload.Arguments, dataset)
	case "query_members":
		return handleQueryMembers(req.ID, payload.Arguments, dataset)
	case "get_project_details":
		return handleGetProjectDetails(req.ID, payload.Arguments, dataset)
	default:
		// Handle metric-based tools
		var args struct {
			Metric string `json:"metric"`
		}
		if len(def.Metrics) > 0 {
			if len(payload.Arguments) > 0 {
				if err := json.Unmarshal(payload.Arguments, &args); err != nil {
					return errorResponse(req.ID, -32602, "Invalid arguments", nil)
				}
			}
			if args.Metric == "" {
				return errorResponse(req.ID, -32602, "Metric is required", mustJSON(map[string]interface{}{
					"allowedMetrics": def.Metrics,
				}))
			}
			if !metricAllowed(def.Metrics, args.Metric) {
				return errorResponse(req.ID, -32602, fmt.Sprintf("Unsupported metric %q", args.Metric), mustJSON(map[string]interface{}{
					"allowedMetrics": def.Metrics,
				}))
			}
		}

		result, err := executeMetric(args.Metric, dataset, time.Now().UTC())
		if err != nil {
			return errorResponse(req.ID, -32000, err.Error(), nil)
		}

		return &jsonRPCResponse{
			JSONRPC: "2.0",
			Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
			ID:      req.ID,
		}
	}
}

func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func toolInputSchema(def toolDefinition) map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
	}

	// Special schemas for new query tools
	switch def.Name {
	case "query_projects":
		schema["properties"] = map[string]interface{}{
			"maturity": map[string]interface{}{
				"type":        "string",
				"description": "Filter by maturity level (e.g., graduated, incubating, sandbox)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Filter by project name (case-insensitive substring match)",
			},
			"graduated_from": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects graduated on or after this date (YYYY-MM-DD)",
			},
			"graduated_to": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects graduated on or before this date (YYYY-MM-DD)",
			},
			"incubating_from": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects that reached incubating on or after this date (YYYY-MM-DD)",
			},
			"incubating_to": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects that reached incubating on or before this date (YYYY-MM-DD)",
			},
			"accepted_from": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects accepted on or after this date (YYYY-MM-DD)",
			},
			"accepted_to": map[string]interface{}{
				"type":        "string",
				"description": "Filter projects accepted on or before this date (YYYY-MM-DD)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 100)",
			},
		}
		return schema
	case "query_members":
		schema["properties"] = map[string]interface{}{
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "Filter by membership tier (e.g., Gold, Silver, Platinum, End User Supporter)",
			},
			"joined_from": map[string]interface{}{
				"type":        "string",
				"description": "Filter members joined on or after this date (YYYY-MM-DD)",
			},
			"joined_to": map[string]interface{}{
				"type":        "string",
				"description": "Filter members joined on or before this date (YYYY-MM-DD)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 100)",
			},
		}
		return schema
	case "get_project_details":
		schema["properties"] = map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Project name to search for (case-insensitive)",
			},
		}
		schema["required"] = []string{"name"}
		return schema
	}

	// Default metric-based schema
	if len(def.Metrics) > 0 {
		schema["properties"] = map[string]interface{}{
			"metric": map[string]interface{}{
				"type": "string",
				"enum": def.Metrics,
			},
		}
		schema["required"] = []string{"metric"}
	} else {
		schema["properties"] = map[string]interface{}{}
	}
	return schema
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

func sendNotification(method string, params interface{}) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}

	outputMu.Lock()
	defer outputMu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("unable to marshal notification %s: %v", method, err)
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

func metricAllowed(allowed []string, metric string) bool {
	for _, m := range allowed {
		if m == metric {
			return true
		}
	}
	return false
}

func handleQueryProjects(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Maturity       string `json:"maturity"`
		Name           string `json:"name"`
		GraduatedFrom  string `json:"graduated_from"`
		GraduatedTo    string `json:"graduated_to"`
		IncubatingFrom string `json:"incubating_from"`
		IncubatingTo   string `json:"incubating_to"`
		AcceptedFrom   string `json:"accepted_from"`
		AcceptedTo     string `json:"accepted_to"`
		Limit          int    `json:"limit"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if args.Limit == 0 {
		args.Limit = 100
	}

	result, err := queryProjects(ds, args.Maturity, args.Name, args.GraduatedFrom, args.GraduatedTo, args.IncubatingFrom, args.IncubatingTo, args.AcceptedFrom, args.AcceptedTo, args.Limit)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleQueryMembers(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Tier       string `json:"tier"`
		JoinedFrom string `json:"joined_from"`
		JoinedTo   string `json:"joined_to"`
		Limit      int    `json:"limit"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if args.Limit == 0 {
		args.Limit = 100
	}

	result, err := queryMembers(ds, args.Tier, args.JoinedFrom, args.JoinedTo, args.Limit)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleGetProjectDetails(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Name string `json:"name"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if args.Name == "" {
		return errorResponse(id, -32602, "name parameter is required", nil)
	}

	result, err := getProjectDetails(ds, args.Name)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}
