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

// serverVersion is the semantic version of the MCP server, reported in the
// initialize handshake.
const serverVersion = "0.2.0"

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
	src     dataSource
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
			"compare_projects": {
				Name:        "compare_projects",
				Description: fmt.Sprintf("Side-by-side comparison of two or more %s projects.", cfg.Name),
			},
			"funding_analysis": {
				Name:        "funding_analysis",
				Description: fmt.Sprintf("Analyze funding data from Crunchbase for %s landscape members.", cfg.Name),
			},
			"geographic_distribution": {
				Name:        "geographic_distribution",
				Description: fmt.Sprintf("Geographic breakdown of %s landscape members by Crunchbase location data.", cfg.Name),
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
			"compare_projects",
			"funding_analysis",
			"geographic_distribution",
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

// refreshDataset reloads the dataset from the configured data source.
// It replaces the in-memory dataset under a write lock so that in-flight
// requests see a consistent snapshot.
func refreshDataset(ctx context.Context, state *serverState) {
	log.Println("dataset refresh requested")
	go func() {
		ds, err := loadDataset(ctx, state.src)
		if err != nil {
			log.Printf("dataset refresh failed: %v", err)
			return
		}
		state.mu.Lock()
		state.dataset = ds
		state.loadErr = nil
		state.mu.Unlock()
		log.Printf("dataset refreshed (%d items)", len(ds.Items))
	}()
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

	src := dataSource{
		filePath: *dataFile,
		url:      *dataURL,
	}
	state.src = src

	go func() {
		ds, err := loadDataset(ctx, src)
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
	case "notifications/dataset_refreshed":
		refreshDataset(ctx, state)
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
	case "resources/list":
		return handleResourcesList(req, state)
	case "resources/read":
		return handleResourcesRead(ctx, req, state)
	case "prompts/list":
		return handlePromptsList(req, state)
	case "prompts/get":
		return handlePromptsGet(req, state)
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
	case "compare_projects":
		return handleCompareProjects(req.ID, payload.Arguments, dataset)
	case "funding_analysis":
		return handleFundingAnalysis(req.ID, payload.Arguments, dataset)
	case "geographic_distribution":
		return handleGeographicDistribution(req.ID, payload.Arguments, dataset)
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
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to skip before returning (default: 0)",
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
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to skip before returning (default: 0)",
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
	case "compare_projects":
		schema["properties"] = map[string]interface{}{
			"names": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Project names to compare (minimum 2)",
				"minItems":    2,
			},
		}
		schema["required"] = []string{"names"}
		return schema
	case "funding_analysis":
		schema["properties"] = map[string]interface{}{
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "Filter by membership tier (e.g. Gold, Silver)",
			},
			"year": map[string]interface{}{
				"type":        "integer",
				"description": "Filter funding rounds to a specific year",
			},
		}
		return schema
	case "geographic_distribution":
		schema["properties"] = map[string]interface{}{
			"group_by": map[string]interface{}{
				"type":        "string",
				"description": "Group by: country (default), region, or city",
				"enum":        []string{"country", "region", "city"},
			},
		}
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
		Offset         int    `json:"offset"`
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
		Offset:         args.Offset,
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
		Offset     int    `json:"offset"`
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
		Offset:     args.Offset,
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

func handleCompareProjects(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Names []string `json:"names"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}
	if len(args.Names) < 2 {
		return errorResponse(id, -32602, "at least 2 project names are required", nil)
	}

	result, err := compareProjects(ds, args.Names)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleFundingAnalysis(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		Tier string `json:"tier"`
		Year int    `json:"year"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}

	result, err := fundingAnalysis(ds, args.Tier, args.Year)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

func handleGeographicDistribution(id json.RawMessage, argsRaw json.RawMessage, ds *Dataset) *jsonRPCResponse {
	var args struct {
		GroupBy string `json:"group_by"`
	}
	if len(argsRaw) > 0 {
		if err := json.Unmarshal(argsRaw, &args); err != nil {
			return errorResponse(id, -32602, "Invalid arguments", nil)
		}
	}

	result, err := geographicDistribution(ds, args.GroupBy)
	if err != nil {
		return errorResponse(id, -32000, err.Error(), nil)
	}

	return &jsonRPCResponse{
		JSONRPC: "2.0",
		Result:  mustJSON(map[string]interface{}{"content": []map[string]string{{"type": "text", "text": result}}}),
		ID:      id,
	}
}

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

	dataset, err := state.waitForDataset(ctx)
	if err != nil {
		return errorResponse(req.ID, -32603, "Dataset unavailable", mustJSON(map[string]string{"error": err.Error()}))
	}

	var text string
	switch params.URI {
	case "landscape://categories":
		text, err = listCategories(dataset)
	case "landscape://summary":
		text, err = landscapeSummary(dataset)
	default:
		return errorResponse(req.ID, -32602, fmt.Sprintf("Unknown resource URI: %s", params.URI), nil)
	}

	if err != nil {
		return errorResponse(req.ID, -32000, err.Error(), nil)
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
3. Identify trends in project maturity levels
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
	default:
		return errorResponse(req.ID, -32602, fmt.Sprintf("Unknown prompt: %s", params.Name), nil)
	}
}
