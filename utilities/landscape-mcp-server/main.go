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
	cfg     LandscapeConfig
	tools   toolCatalogData
}

var outputMu sync.Mutex

type toolDefinition struct {
	Name        string
	Description string
	Metrics     []string
}

type toolCatalogData struct {
	catalog         map[string]toolDefinition
	advertisedTools []string
}

func buildToolCatalog(cfg LandscapeConfig) toolCatalogData {
	return toolCatalogData{
		catalog: map[string]toolDefinition{
			"project_metrics": {
				Name:        "project_metrics",
				Description: fmt.Sprintf("Answer questions about %s project counts and progress.", cfg.Name),
				Metrics: []string{
					"incubating_project_count",
					"sandbox_projects_joined_this_year",
					"projects_graduated_last_year",
				},
			},
			"membership_metrics": {
				Name:        "membership_metrics",
				Description: fmt.Sprintf("Answer questions about %s membership activity and funding.", cfg.Name),
				Metrics: []string{
					"gold_members_joined_this_year",
					"silver_members_joined_this_year",
					"silver_members_raised_last_month",
					"gold_members_raised_this_year",
				},
			},
			"query_projects": {
				Name:        "query_projects",
				Description: fmt.Sprintf("Query %s projects with flexible filtering by maturity, dates, and name. Returns detailed project information.", cfg.Name),
			},
			"query_members": {
				Name:        "query_members",
				Description: fmt.Sprintf("Query %s members with flexible filtering by membership tier and join dates.", cfg.Name),
			},
			"get_project_details": {
				Name:        "get_project_details",
				Description: fmt.Sprintf("Get detailed information about a specific %s project by name.", cfg.Name),
			},
			"get_member_details": {
				Name:        "get_member_details",
				Description: fmt.Sprintf("Get detailed information about a specific %s member by name.", cfg.Name),
			},
			"list_categories": {
				Name:        "list_categories",
				Description: fmt.Sprintf("List all categories and subcategories in the %s landscape with item counts.", cfg.Name),
			},
			"landscape_summary": {
				Name:        "landscape_summary",
				Description: fmt.Sprintf("Get a high-level statistical overview of the entire %s landscape.", cfg.Name),
			},
			"search_landscape": {
				Name:        "search_landscape",
				Description: fmt.Sprintf("Full-text search across all %s landscape items by name, description, category, subcategory, or homepage URL.", cfg.Name),
			},
		},
		advertisedTools: []string{
			"query_projects",
			"query_members",
			"get_project_details",
			"get_member_details",
			"list_categories",
			"landscape_summary",
			"search_landscape",
			"project_metrics",
			"membership_metrics",
		},
	}
}

func newServerState(cfg LandscapeConfig) *serverState {
	return &serverState{
		ready: make(chan struct{}),
		cfg:   cfg,
		tools: buildToolCatalog(cfg),
	}
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
	landscapeName := flag.String("landscape-name", "", "Name of the landscape (e.g. CNCF)")
	landscapeDesc := flag.String("landscape-description", "", "Description of the landscape")
	flag.Parse()

	cfg := defaultConfig()
	if *landscapeName != "" {
		cfg.Name = *landscapeName
	}
	if *landscapeDesc != "" {
		cfg.Description = *landscapeDesc
	}

	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	state := newServerState(cfg)
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
					"name":    fmt.Sprintf("%s-landscape-mcp-server", strings.ToLower(state.cfg.Name)),
					"version": "0.2.0",
				},
			}),
			ID: req.ID,
		}
	case "notifications/initialized":
		return nil
	case "tools/list":
		tools := make([]map[string]interface{}, 0, len(state.tools.advertisedTools))
		for _, name := range state.tools.advertisedTools {
			def, ok := state.tools.catalog[name]
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

	def, ok := state.tools.catalog[payload.Name]
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
	case "get_member_details":
		return handleGetMemberDetails(req.ID, payload.Arguments, dataset)
	case "list_categories":
		return handleListCategories(req.ID, dataset)
	case "landscape_summary":
		return handleLandscapeSummary(req.ID, dataset)
	case "search_landscape":
		return handleSearchLandscape(req.ID, payload.Arguments, dataset)
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
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter by category (e.g., Provisioning, Runtime, Observability and Analysis)",
			},
			"subcategory": map[string]interface{}{
				"type":        "string",
				"description": "Filter by subcategory (e.g., Container Runtime, Service Mesh)",
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
			"category": map[string]interface{}{
				"type":        "string",
				"description": "Filter by category (e.g., CNCF Members)",
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
	case "get_member_details":
		schema["properties"] = map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Member name to search for (case-insensitive)",
			},
		}
		schema["required"] = []string{"name"}
		return schema
	case "list_categories":
		// No additional properties needed
		return schema
	case "landscape_summary":
		// No additional properties needed
		return schema
	case "search_landscape":
		schema["properties"] = map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search term (case-insensitive substring match)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum results (default 20, max 100)",
			},
		}
		schema["required"] = []string{"query"}
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
		Category       string `json:"category"`
		Subcategory    string `json:"subcategory"`
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
	if args.Limit <= 0 || args.Limit > 500 {
		args.Limit = 100
	}

	result, err := queryProjects(ds, ProjectQuery{
		Maturity:       args.Maturity,
		Name:           args.Name,
		Category:       args.Category,
		Subcategory:    args.Subcategory,
		GraduatedFrom:  args.GraduatedFrom,
		GraduatedTo:    args.GraduatedTo,
		IncubatingFrom: args.IncubatingFrom,
		IncubatingTo:   args.IncubatingTo,
		AcceptedFrom:   args.AcceptedFrom,
		AcceptedTo:     args.AcceptedTo,
		Limit:          args.Limit,
	})
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
		Category   string `json:"category"`
		JoinedFrom string `json:"joined_from"`
		JoinedTo   string `json:"joined_to"`
		Limit      int    `json:"limit"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if args.Limit <= 0 || args.Limit > 500 {
		args.Limit = 100
	}

	result, err := queryMembers(ds, MemberQuery{
		Tier:       args.Tier,
		Category:   args.Category,
		JoinedFrom: args.JoinedFrom,
		JoinedTo:   args.JoinedTo,
		Limit:      args.Limit,
	})
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

func handleGetMemberDetails(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
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

	result, err := getMemberDetails(ds, args.Name)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleListCategories(id json.RawMessage, ds *Dataset) *jsonRPCResponse {
	result, err := listCategories(ds)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleLandscapeSummary(id json.RawMessage, ds *Dataset) *jsonRPCResponse {
	result, err := landscapeSummary(ds)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleSearchLandscape(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if args.Query == "" {
		return errorResponse(id, -32602, "query parameter is required", nil)
	}

	result, err := searchLandscape(ds, args.Query, args.Limit)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}
