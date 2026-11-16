package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

type toolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func newTestState(t *testing.T, ds *Dataset) *serverState {
	t.Helper()
	state := newServerState(LandscapeConfig{Name: "CNCF", Description: "test"})
	state.setDataset(ds, nil)
	return state
}

func callTool(t *testing.T, state *serverState, name string, args string) *jsonRPCResponse {
	t.Helper()
	params := fmt.Sprintf(`{"name":%q,"arguments":%s}`, name, args)
	resp := handleRequest(context.Background(), &jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(params),
		ID:      json.RawMessage(`1`),
	}, state)
	if resp == nil {
		t.Fatalf("tools/call %s returned no response", name)
	}
	return resp
}

func decodeToolResult(t *testing.T, resp *jsonRPCResponse) toolResult {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %+v", resp.Error)
	}
	var res toolResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		t.Fatalf("failed to decode tool result: %v", err)
	}
	if len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("unexpected content shape: %s", resp.Result)
	}
	return res
}

func ambiguityDataset() *Dataset {
	return &Dataset{Items: []LandscapeItem{
		makeItem("Foo", "graduated"),
		makeItem("Foo Beta", "sandbox"),
		makeItem("Alpha One", "sandbox"),
		makeItem("Alpha Two", "sandbox"),
		makeItem("Solo", "incubating"),
	}}
}

func TestNotificationsGetNoResponse(t *testing.T) {
	state := newTestState(t, loadTestDataset(t))

	tests := []struct {
		name   string
		method string
		params string
	}{
		{"initialized", "notifications/initialized", ""},
		{"unknown notification", "notifications/does_not_exist", ""},
		{"unknown non-notification-prefixed method", "bogus/method", ""},
		{"initialize without id", "initialize", ""},
		{"tools/call without id", "tools/call", `{"name":"nope"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a message without an id
			req := &jsonRPCRequest{JSONRPC: "2.0", Method: tt.method}
			if tt.params != "" {
				req.Params = json.RawMessage(tt.params)
			}
			// When it is handled
			resp := handleRequest(context.Background(), req, state)
			// Then no response is produced
			if resp != nil {
				t.Errorf("expected no response for notification, got %+v", resp)
			}
		})
	}
}

func TestUnknownMethodWithIDReturnsMethodNotFound(t *testing.T) {
	state := newTestState(t, loadTestDataset(t))
	resp := handleRequest(context.Background(), &jsonRPCRequest{
		JSONRPC: "2.0", Method: "bogus/method", ID: json.RawMessage(`7`),
	}, state)
	if resp == nil || resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected -32601, got %+v", resp)
	}
}

func TestToolsCallProtocolErrors(t *testing.T) {
	state := newTestState(t, loadTestDataset(t))

	tests := []struct {
		name     string
		params   string
		wantCode int
	}{
		{"unknown tool", `{"name":"no_such_tool","arguments":{}}`, -32602},
		{"params wrong shape", `[1,2,3]`, -32602},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := handleRequest(context.Background(), &jsonRPCRequest{
				JSONRPC: "2.0", Method: "tools/call",
				Params: json.RawMessage(tt.params), ID: json.RawMessage(`1`),
			}, state)
			if resp == nil || resp.Error == nil {
				t.Fatalf("expected JSON-RPC error, got %+v", resp)
			}
			if resp.Error.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", resp.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestToolExecutionFailuresAreIsErrorResults(t *testing.T) {
	state := newTestState(t, ambiguityDataset())

	tests := []struct {
		name     string
		tool     string
		args     string
		wantText string
	}{
		{"project not found", "get_project_details", `{"name":"zzz-missing"}`, "no project found"},
		{"bad date", "query_projects", `{"graduated_from":"not-a-date"}`, "invalid graduated_from"},
		{"missing required arg", "get_project_details", `{}`, "name parameter is required"},
		{"wrong arg type", "query_projects", `{"limit":"ten"}`, "Invalid arguments"},
		{"unsupported metric", "project_metrics", `{"metric":"bogus"}`, "Unsupported metric"},
		{"missing metric", "project_metrics", `{}`, "Metric is required"},
		{"compare too few", "compare_projects", `{"names":["Foo"]}`, "at least 2"},
		{"compare unknown", "compare_projects", `{"names":["Foo","zzz-missing"]}`, "no project found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := decodeToolResult(t, callTool(t, state, tt.tool, tt.args))
			if !res.IsError {
				t.Errorf("isError = false, want true; text=%q", res.Content[0].Text)
			}
			if !strings.Contains(res.Content[0].Text, tt.wantText) {
				t.Errorf("text = %q, want it to contain %q", res.Content[0].Text, tt.wantText)
			}
		})
	}
}

func TestSuccessfulToolResultIsNotError(t *testing.T) {
	state := newTestState(t, ambiguityDataset())
	res := decodeToolResult(t, callTool(t, state, "get_project_details", `{"name":"Solo"}`))
	if res.IsError {
		t.Errorf("isError = true for successful call: %s", res.Content[0].Text)
	}
}

func TestProjectNameResolution(t *testing.T) {
	state := newTestState(t, ambiguityDataset())

	tests := []struct {
		name       string
		tool       string
		args       string
		wantError  bool
		wantInText []string
	}{
		{"details exact match beats substring", "get_project_details", `{"name":"foo"}`, false, []string{`"count": 1`}},
		{"details unique substring", "get_project_details", `{"name":"sol"}`, false, []string{"Solo"}},
		{"details ambiguous substring lists candidates", "get_project_details", `{"name":"alpha"}`, true, []string{"Alpha One", "Alpha Two"}},
		{"compare exact match beats substring", "compare_projects", `{"names":["foo","solo"]}`, false, []string{`"count": 2`}},
		{"compare ambiguous name lists candidates", "compare_projects", `{"names":["Solo","alpha"]}`, true, []string{"Alpha One", "Alpha Two"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := decodeToolResult(t, callTool(t, state, tt.tool, tt.args))
			if res.IsError != tt.wantError {
				t.Fatalf("isError = %v, want %v; text=%q", res.IsError, tt.wantError, res.Content[0].Text)
			}
			for _, want := range tt.wantInText {
				if !strings.Contains(res.Content[0].Text, want) {
					t.Errorf("text %q does not contain %q", res.Content[0].Text, want)
				}
			}
		})
	}

	t.Run("blank name is rejected instead of matching everything", func(t *testing.T) {
		for _, tool := range []string{"get_project_details", "get_member_details"} {
			res := decodeToolResult(t, callTool(t, state, tool, `{"name":"  "}`))
			if !res.IsError || strings.Contains(res.Content[0].Text, "Alpha") {
				t.Errorf("%s: want an error naming no candidates, got %+v", tool, res)
			}
		}
	})

	t.Run("exact match does not return substring neighbors", func(t *testing.T) {
		res := decodeToolResult(t, callTool(t, state, "get_project_details", `{"name":"foo"}`))
		if strings.Contains(res.Content[0].Text, "Foo Beta") {
			t.Errorf("exact match leaked substring match: %s", res.Content[0].Text)
		}
	})
}

func TestBlankLookupNamesMatchNothing(t *testing.T) {
	ds := &Dataset{Items: []LandscapeItem{
		makeItem("Alpha One", "sandbox"),
		makeItem("Alpha Two", "sandbox"),
		makeItem("Acme Corp (member)", "", func(i *LandscapeItem) { i.Category = "CNCF Members" }),
		makeItem("Beta Inc (member)", "", func(i *LandscapeItem) { i.Category = "CNCF Members" }),
	}}

	for _, name := range []string{"", " ", "\t "} {
		if found, err := resolveProjects(ds, name); err == nil || strings.Contains(err.Error(), "Alpha") {
			t.Errorf("resolveProjects(%q) = %d projects, err %v; want an error naming no candidates", name, len(found), err)
		}
		if text, err := getMemberDetails(ds, name); err == nil {
			t.Errorf("getMemberDetails(%q) returned members: %s", name, text)
		}
	}
}

func archivedDataset() *Dataset {
	return &Dataset{Items: []LandscapeItem{
		makeItem("LiveInc", "incubating", func(i *LandscapeItem) { i.Category = "Runtime" }),
		makeItem("OldInc", "incubating", func(i *LandscapeItem) {
			i.Category = "Runtime"
			i.ArchivedAt = timePtr("2024-01-01")
		}),
		makeItem("OldSandbox", "sandbox", func(i *LandscapeItem) {
			i.AcceptedAt = timePtr("2026-02-01")
			i.ArchivedAt = timePtr("2026-03-01")
		}),
		makeItem("OldGrad", "graduated", func(i *LandscapeItem) {
			i.GraduatedAt = timePtr("2025-05-01")
			i.ArchivedAt = timePtr("2026-03-01")
		}),
		makeItem("OldMember", "", func(i *LandscapeItem) {
			i.Category = "CNCF Members"
			i.Subcategory = "Gold"
			i.ArchivedAt = timePtr("2026-03-01")
		}),
	}}
}

func TestArchivedItemsExcludedFromAggregates(t *testing.T) {
	ds := archivedDataset()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	metrics := []struct {
		metric string
		want   int
	}{
		{"incubating_project_count", 1},
		{"sandbox_projects_joined_this_year", 0},
		{"projects_graduated_last_year", 0},
	}
	for _, tt := range metrics {
		t.Run(tt.metric, func(t *testing.T) {
			out, err := executeMetric(tt.metric, ds, now, defaultConfig())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := jsonInt(t, out, "value"); got != tt.want {
				t.Errorf("value = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("landscape_summary totals", func(t *testing.T) {
		out, err := landscapeSummary(ds)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := jsonInt(t, out, "total_items"); got != 1 {
			t.Errorf("total_items = %d, want 1", got)
		}
		if got := jsonInt(t, out, "total_projects"); got != 1 {
			t.Errorf("total_projects = %d, want 1", got)
		}
		if got := jsonInt(t, out, "total_members"); got != 0 {
			t.Errorf("total_members = %d, want 0", got)
		}
	})

	t.Run("query tools exclude archived", func(t *testing.T) {
		state := newTestState(t, ds)
		projects := decodeToolResult(t, callTool(t, state, "query_projects", `{}`))
		if got := jsonInt(t, projects.Content[0].Text, "total"); got != 1 {
			t.Errorf("query_projects total = %d, want 1", got)
		}
		members := decodeToolResult(t, callTool(t, state, "query_members", `{}`))
		if got := jsonInt(t, members.Content[0].Text, "total"); got != 0 {
			t.Errorf("query_members total = %d, want 0", got)
		}
	})

	t.Run("direct lookups still find archived", func(t *testing.T) {
		state := newTestState(t, ds)
		res := decodeToolResult(t, callTool(t, state, "get_project_details", `{"name":"OldInc"}`))
		if res.IsError || !strings.Contains(res.Content[0].Text, "OldInc") {
			t.Errorf("expected OldInc details, got %+v", res)
		}
		res = decodeToolResult(t, callTool(t, state, "get_member_details", `{"name":"OldMember"}`))
		if res.IsError {
			t.Errorf("expected OldMember details, got %+v", res)
		}
		res = decodeToolResult(t, callTool(t, state, "search_landscape", `{"query":"OldInc"}`))
		if res.IsError || !strings.Contains(res.Content[0].Text, "OldInc") {
			t.Errorf("expected search hit for OldInc, got %+v", res)
		}
	})
}

func manyProjects(n int) *Dataset {
	items := make([]LandscapeItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, makeItem(fmt.Sprintf("proj-%04d", i), "sandbox"))
	}
	return &Dataset{Items: items}
}

func manyMembers(n int) *Dataset {
	items := make([]LandscapeItem, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, makeItem(fmt.Sprintf("member-%04d", i), "", func(it *LandscapeItem) {
			it.Category = "CNCF Members"
			it.Subcategory = "Silver"
		}))
	}
	return &Dataset{Items: items}
}

func TestQueryLimitClampAndTotal(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		ds        *Dataset
		args      string
		wantCount int
		wantTotal int
	}{
		{"projects limit above max clamps to 500", "query_projects", manyProjects(600), `{"limit":1000}`, 500, 600},
		{"projects limit zero defaults to 100", "query_projects", manyProjects(600), `{"limit":0}`, 100, 600},
		{"projects negative limit defaults to 100", "query_projects", manyProjects(600), `{"limit":-3}`, 100, 600},
		{"projects explicit small limit honored", "query_projects", manyProjects(600), `{"limit":7}`, 7, 600},
		{"projects total ignores limit and offset", "query_projects", manyProjects(30), `{"limit":5,"offset":10}`, 5, 30},
		{"projects offset past end keeps total", "query_projects", manyProjects(30), `{"offset":100}`, 0, 30},
		{"members limit above max clamps to 500", "query_members", manyMembers(600), `{"limit":501}`, 500, 600},
		{"members limit zero defaults to 100", "query_members", manyMembers(600), `{}`, 100, 600},
		{"members total ignores limit and offset", "query_members", manyMembers(30), `{"limit":5,"offset":28}`, 2, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState(t, tt.ds)
			res := decodeToolResult(t, callTool(t, state, tt.tool, tt.args))
			if res.IsError {
				t.Fatalf("unexpected tool error: %s", res.Content[0].Text)
			}
			if got := jsonInt(t, res.Content[0].Text, "count"); got != tt.wantCount {
				t.Errorf("count = %d, want %d", got, tt.wantCount)
			}
			if got := jsonInt(t, res.Content[0].Text, "total"); got != tt.wantTotal {
				t.Errorf("total = %d, want %d", got, tt.wantTotal)
			}
		})
	}
}

func TestToolsCallDatasetTimeoutIsErrorResult(t *testing.T) {
	// Given a server whose dataset never loads and a short wait budget
	old := datasetWaitTimeout
	datasetWaitTimeout = 20 * time.Millisecond
	t.Cleanup(func() { datasetWaitTimeout = old })
	state := newServerState(LandscapeConfig{Name: "CNCF"})

	// When a tool is called
	res := decodeToolResult(t, callTool(t, state, "landscape_summary", `{}`))

	// Then it fails as an isError result mentioning the timeout
	if !res.IsError || !strings.Contains(res.Content[0].Text, "timed out") {
		t.Errorf("expected timeout isError result, got %+v", res)
	}
}

func TestResourcesReadDatasetTimeout(t *testing.T) {
	old := datasetWaitTimeout
	datasetWaitTimeout = 20 * time.Millisecond
	t.Cleanup(func() { datasetWaitTimeout = old })
	state := newServerState(LandscapeConfig{Name: "CNCF"})

	resp := handleRequest(context.Background(), &jsonRPCRequest{
		JSONRPC: "2.0", Method: "resources/read",
		Params: json.RawMessage(`{"uri":"landscape://summary"}`), ID: json.RawMessage(`1`),
	}, state)
	if resp == nil || resp.Error == nil || !strings.Contains(resp.Error.Message, "timed out") {
		t.Fatalf("expected timeout error, got %+v", resp)
	}
}

func TestDecodeRequestRejectsUnservableLines(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantCode int
		wantID   string
	}{
		{"not json", `this is not json`, -32700, "null"},
		{"truncated json", `{"jsonrpc":"2.0","id":1,"method":"pi`, -32700, "null"},
		{"batch array", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, -32600, "null"},
		{"scalar", `5`, -32600, "null"},
		{"wrong field type keeps the id", `{"jsonrpc":"2.0","id":5,"method":123}`, -32600, "5"},
		{"wrong field type with string id", `{"jsonrpc":"2.0","id":"abc","method":[]}`, -32600, `"abc"`},
		{"wrong field type without id", `{"jsonrpc":"2.0","method":123}`, -32600, "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When an unservable line is decoded
			req, resp := decodeRequest([]byte(tt.line))

			// Then it yields an error response instead of a request
			if req != nil || resp == nil || resp.Error == nil {
				t.Fatalf("req = %+v, resp = %+v; want only an error response", req, resp)
			}
			if resp.Error.Code != tt.wantCode {
				t.Errorf("code = %d, want %d", resp.Error.Code, tt.wantCode)
			}
			// And the id survives marshalling, including an explicit null
			var wire map[string]json.RawMessage
			if err := json.Unmarshal(mustJSON(resp), &wire); err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			if got := string(wire["id"]); got != tt.wantID {
				t.Errorf("id on the wire = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestDecodeRequestAcceptsRequestsAndNotifications(t *testing.T) {
	tests := []struct {
		name, line, wantMethod, wantID string
	}{
		{"request", `{"jsonrpc":"2.0","id":7,"method":"ping"}`, "ping", "7"},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, "ping", "null"},
		{"notification", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "notifications/initialized", ""},
		{"leading whitespace", `  {"jsonrpc":"2.0","id":1,"method":"ping"}`, "ping", "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, resp := decodeRequest([]byte(tt.line))

			if resp != nil || req == nil {
				t.Fatalf("req = %+v, resp = %+v; want only a request", req, resp)
			}
			if req.Method != tt.wantMethod || string(req.ID) != tt.wantID {
				t.Errorf("method/id = %q/%q, want %q/%q", req.Method, req.ID, tt.wantMethod, tt.wantID)
			}
		})
	}
}

func TestReadLine(t *testing.T) {
	type line struct {
		text    string
		tooLong bool
	}
	tests := []struct {
		name  string
		input string
		max   int
		want  []line
	}{
		{"normal lines", "abc\ndef\n", 10, []line{{"abc\n", false}, {"def\n", false}}},
		{"unterminated final line", "abc\ndef", 10, []line{{"abc\n", false}, {"def", false}}},
		{"oversized line skipped, next line kept", strings.Repeat("x", 100000) + "\nok\n", 50, []line{{"", true}, {"ok\n", false}}},
		{"line spanning buffer size within max", strings.Repeat("y", 5000) + "\n", 10000, []line{{strings.Repeat("y", 5000) + "\n", false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReaderSize(strings.NewReader(tt.input), 16)
			for i, want := range tt.want {
				got, tooLong, err := readLine(r, tt.max)
				if err != nil {
					t.Fatalf("line %d: unexpected error %v", i, err)
				}
				if tooLong != want.tooLong || (!tooLong && string(got) != want.text) {
					t.Errorf("line %d: got (%d bytes, tooLong=%v), want (%d bytes, tooLong=%v)", i, len(got), tooLong, len(want.text), want.tooLong)
				}
			}
			if _, _, err := readLine(r, tt.max); err == nil {
				t.Error("expected EOF after last line")
			}
		})
	}
}
