package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func rpc(t *testing.T, state *serverState, method, params string, id string) *jsonRPCResponse {
	t.Helper()
	req := &jsonRPCRequest{JSONRPC: "2.0", Method: method}
	if params != "" {
		req.Params = json.RawMessage(params)
	}
	if id != "" {
		req.ID = json.RawMessage(id)
	}
	return handleRequest(context.Background(), req, state)
}

func mustResult(t *testing.T, resp *jsonRPCResponse) map[string]json.RawMessage {
	t.Helper()
	if resp == nil {
		t.Fatal("expected a response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %+v", resp.Error)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(resp.Result, &out); err != nil {
		t.Fatalf("result is not an object: %v (%s)", err, resp.Result)
	}
	return out
}

func complianceDataset() *Dataset {
	return &Dataset{Items: []LandscapeItem{
		makeItem("Envoy", "graduated"),
		makeItem("Alpha One", "sandbox"),
		makeItem("Alpha Two", "sandbox"),
	}}
}

func TestPingReturnsEmptyResultWithID(t *testing.T) {
	state := newTestState(t, complianceDataset())

	resp := rpc(t, state, "ping", "", "7")

	if resp == nil || resp.Error != nil {
		t.Fatalf("expected success response, got %+v", resp)
	}
	if string(resp.Result) != "{}" || string(resp.ID) != "7" {
		t.Fatalf("expected result {} with id 7, got result=%s id=%s", resp.Result, resp.ID)
	}
}

func TestPingNotificationGetsNoResponse(t *testing.T) {
	state := newTestState(t, complianceDataset())

	if resp := rpc(t, state, "ping", "", ""); resp != nil {
		t.Fatalf("expected no response, got %+v", resp)
	}
}

func TestInitializeProtocolVersionNegotiation(t *testing.T) {
	tests := []struct {
		name     string
		params   string
		wantEcho string
	}{
		{"oldest supported is echoed", `{"protocolVersion":"2024-11-05"}`, "2024-11-05"},
		{"middle supported is echoed", `{"protocolVersion":"2025-03-26"}`, "2025-03-26"},
		{"latest supported is echoed", `{"protocolVersion":"2025-06-18"}`, "2025-06-18"},
		{"unknown version gets latest", `{"protocolVersion":"1999-01-01"}`, "2025-06-18"},
		{"missing params gets latest", ``, "2025-06-18"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newTestState(t, complianceDataset())

			res := mustResult(t, rpc(t, state, "initialize", tt.params, "1"))

			var got string
			if err := json.Unmarshal(res["protocolVersion"], &got); err != nil {
				t.Fatalf("protocolVersion: %v", err)
			}
			if got != tt.wantEcho {
				t.Errorf("protocolVersion = %q, want %q", got, tt.wantEcho)
			}
			var caps map[string]map[string]interface{}
			if err := json.Unmarshal(res["capabilities"], &caps); err != nil {
				t.Fatalf("capabilities: %v", err)
			}
			for _, k := range []string{"tools", "resources", "prompts"} {
				if _, ok := caps[k]; !ok {
					t.Errorf("capabilities missing %q", k)
				}
			}
		})
	}
}

func TestToolsListEntriesHaveAnnotations(t *testing.T) {
	state := newTestState(t, complianceDataset())

	res := mustResult(t, rpc(t, state, "tools/list", "", "1"))

	var tools []struct {
		Name        string          `json:"name"`
		InputSchema json.RawMessage `json:"inputSchema"`
		Annotations map[string]bool `json:"annotations"`
	}
	if err := json.Unmarshal(res["tools"], &tools); err != nil {
		t.Fatalf("tools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("no tools listed")
	}
	want := map[string]bool{"readOnlyHint": true, "destructiveHint": false, "idempotentHint": true, "openWorldHint": false}
	for _, tool := range tools {
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %s lost its inputSchema", tool.Name)
		}
		if len(tool.Annotations) != len(want) {
			t.Errorf("tool %s annotations = %v, want %v", tool.Name, tool.Annotations, want)
			continue
		}
		for k, v := range want {
			got, ok := tool.Annotations[k]
			if !ok || got != v {
				t.Errorf("tool %s annotation %s = %v (present=%v), want %v", tool.Name, k, got, ok, v)
			}
		}
	}
}

func TestResourceTemplatesList(t *testing.T) {
	state := newTestState(t, complianceDataset())

	res := mustResult(t, rpc(t, state, "resources/templates/list", "", "1"))

	var templates []map[string]string
	if err := json.Unmarshal(res["resourceTemplates"], &templates); err != nil {
		t.Fatalf("resourceTemplates: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d: %v", len(templates), templates)
	}
	wantURIs := []string{"landscape://project/{name}", "landscape://member/{name}"}
	for i, tpl := range templates {
		if tpl["uriTemplate"] != wantURIs[i] {
			t.Errorf("template %d uriTemplate = %q, want %q", i, tpl["uriTemplate"], wantURIs[i])
		}
		if tpl["name"] == "" || tpl["description"] == "" || tpl["mimeType"] != "application/json" {
			t.Errorf("template %d incomplete: %v", i, tpl)
		}
	}
}

func TestResourcesListUnchanged(t *testing.T) {
	state := newTestState(t, complianceDataset())

	res := mustResult(t, rpc(t, state, "resources/list", "", "1"))

	var resources []map[string]string
	if err := json.Unmarshal(res["resources"], &resources); err != nil {
		t.Fatalf("resources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected the 2 static resources, got %d", len(resources))
	}
}

func TestResourcesReadEntityTemplates(t *testing.T) {
	ds := &Dataset{
		Items: []LandscapeItem{
			makeItem("Envoy", "graduated"),
			makeItem("Alpha One", "sandbox"),
			makeItem("Alpha Two", "sandbox"),
			makeItem("OPA/Gatekeeper", "graduated"),
			makeItem("Core 24/7 (member)", "", func(i *LandscapeItem) { i.Category = "CNCF Members" }),
		},
	}
	state := newTestState(t, ds)

	tests := []struct {
		name     string
		uri      string
		wantText string
	}{
		{"project", "landscape://project/Envoy", "Envoy"},
		{"percent-encoded project", "landscape://project/Alpha%20One", "Alpha One"},
		{"encoded slash in project name", "landscape://project/OPA%2FGatekeeper", "OPA/Gatekeeper"},
		{"encoded slash in member name", "landscape://member/Core%2024%2F7%20%28member%29", "Core 24/7 (member)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := mustResult(t, rpc(t, state, "resources/read", `{"uri":"`+tt.uri+`"}`, "1"))

			var contents []map[string]string
			if err := json.Unmarshal(res["contents"], &contents); err != nil {
				t.Fatalf("contents: %v", err)
			}
			if len(contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(contents))
			}
			if contents[0]["uri"] != tt.uri || contents[0]["mimeType"] != "application/json" {
				t.Errorf("unexpected content metadata: %v", contents[0])
			}
			if !strings.Contains(contents[0]["text"], tt.wantText) {
				t.Errorf("text %q does not contain %q", contents[0]["text"], tt.wantText)
			}
		})
	}
}

func TestResourcesReadMemberTemplate(t *testing.T) {
	state := newTestState(t, loadTestDataset(t))

	res := mustResult(t, rpc(t, state, "resources/read", `{"uri":"landscape://member/TestGoldCorp"}`, "1"))

	var contents []map[string]string
	if err := json.Unmarshal(res["contents"], &contents); err != nil {
		t.Fatalf("contents: %v", err)
	}
	if len(contents) != 1 || !strings.Contains(contents[0]["text"], "TestGoldCorp") {
		t.Fatalf("unexpected contents: %v", contents)
	}
}

func TestResourcesReadErrorCodes(t *testing.T) {
	state := newTestState(t, complianceDataset())

	tests := []struct {
		name     string
		uri      string
		wantCode int
	}{
		{"unknown project", "landscape://project/DoesNotExist", -32002},
		{"unknown member", "landscape://member/DoesNotExist", -32002},
		{"unknown resource", "landscape://nope", -32002},
		{"ambiguous project", "landscape://project/Alpha", -32602},
		{"empty name", "landscape://project/", -32602},
		{"blank project name", "landscape://project/%20", -32602},
		{"blank member name", "landscape://member/%20", -32602},
		{"bad percent-encoding", "landscape://project/%zz", -32602},
		{"nested path", "landscape://project/a/b", -32602},
		{"missing uri", "", -32602},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := rpc(t, state, "resources/read", `{"uri":"`+tt.uri+`"}`, "1")

			if resp == nil || resp.Error == nil {
				t.Fatalf("expected error, got %+v", resp)
			}
			if resp.Error.Code != tt.wantCode || resp.Error.Message == "" {
				t.Errorf("error = %+v, want code %d with a message", resp.Error, tt.wantCode)
			}
		})
	}
}

func TestResourcesReadNotFoundCarriesURI(t *testing.T) {
	state := newTestState(t, complianceDataset())

	resp := rpc(t, state, "resources/read", `{"uri":"landscape://project/DoesNotExist"}`, "1")

	if resp == nil || resp.Error == nil {
		t.Fatalf("expected error, got %+v", resp)
	}
	var data map[string]string
	if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
		t.Fatalf("error data: %v", err)
	}
	if data["uri"] != "landscape://project/DoesNotExist" {
		t.Errorf("error data = %v, want the requested uri", data)
	}
}

func TestPromptsListIncludesChangesReport(t *testing.T) {
	state := newTestState(t, complianceDataset())

	res := mustResult(t, rpc(t, state, "prompts/list", "", "1"))

	var prompts []struct {
		Name      string `json:"name"`
		Arguments []struct {
			Name     string `json:"name"`
			Required bool   `json:"required"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(res["prompts"], &prompts); err != nil {
		t.Fatalf("prompts: %v", err)
	}
	if len(prompts) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(prompts))
	}
	for _, p := range prompts {
		if p.Name != "landscape_changes_report" {
			continue
		}
		if len(p.Arguments) != 1 || p.Arguments[0].Name != "since" || p.Arguments[0].Required {
			t.Fatalf("unexpected arguments: %+v", p.Arguments)
		}
		return
	}
	t.Fatal("landscape_changes_report not listed")
}

func promptText(t *testing.T, resp *jsonRPCResponse) string {
	t.Helper()
	res := mustResult(t, resp)
	var msgs []struct {
		Role    string            `json:"role"`
		Content map[string]string `json:"content"`
	}
	if err := json.Unmarshal(res["messages"], &msgs); err != nil {
		t.Fatalf("messages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Role != "user" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	return msgs[0].Content["text"]
}

func TestPromptsGetChangesReport(t *testing.T) {
	state := newTestState(t, complianceDataset())

	t.Run("explicit since is used", func(t *testing.T) {
		text := promptText(t, rpc(t, state, "prompts/get", `{"name":"landscape_changes_report","arguments":{"since":"2024-03-01"}}`, "1"))

		for _, want := range []string{"graduated_from=2024-03-01", "incubating_from=2024-03-01", "accepted_from=2024-03-01", "query_projects"} {
			if !strings.Contains(text, want) {
				t.Errorf("prompt missing %q:\n%s", want, text)
			}
		}
	})

	t.Run("defaults to a date when since is omitted", func(t *testing.T) {
		text := promptText(t, rpc(t, state, "prompts/get", `{"name":"landscape_changes_report"}`, "1"))

		if !strings.Contains(text, "graduated_from=20") {
			t.Errorf("expected a defaulted YYYY-MM-DD date in prompt:\n%s", text)
		}
	})

	t.Run("malformed since is invalid params", func(t *testing.T) {
		resp := rpc(t, state, "prompts/get", `{"name":"landscape_changes_report","arguments":{"since":"last year"}}`, "1")

		if resp == nil || resp.Error == nil || resp.Error.Code != -32602 {
			t.Fatalf("expected -32602, got %+v", resp)
		}
	})
}

func TestAnalyzeLandscapePromptMentionsQueryProjects(t *testing.T) {
	state := newTestState(t, complianceDataset())

	text := promptText(t, rpc(t, state, "prompts/get", `{"name":"analyze_landscape"}`, "1"))

	if !strings.Contains(text, "query_projects") {
		t.Errorf("analyze_landscape prompt does not reference query_projects:\n%s", text)
	}
}

func TestCompareProjectsPromptStillRequiresArgument(t *testing.T) {
	state := newTestState(t, complianceDataset())

	resp := rpc(t, state, "prompts/get", `{"name":"compare_projects"}`, "1")

	if resp == nil || resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected -32602, got %+v", resp)
	}
}

func TestPromptsGetUnknownPromptIsInvalidParams(t *testing.T) {
	// Given a server state
	state := newTestState(t, complianceDataset())

	// When requesting a prompt that does not exist
	resp := rpc(t, state, "prompts/get", `{"name":"no_such_prompt"}`, "1")

	// Then a -32602 error is returned
	if resp == nil || resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected -32602, got %+v", resp)
	}
}
