package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// newGraphQLServer serves the given JSON responses in order, one per request.
func newGraphQLServer(t *testing.T, responses []string) *httptest.Server {
	t.Helper()
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if i >= len(responses) {
			t.Errorf("unexpected request #%d", i+1)
			http.Error(w, "too many requests", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(responses[i])); err != nil {
			t.Errorf("writing response: %v", err)
		}
		i++
	}))
}

func TestListEnterpriseOrgsSkipsBlockedOrgs(t *testing.T) {
	// Orgs that forbid the token come back as null nodes plus a top-level
	// error, alongside the accessible orgs (partial data).
	resp := `{
		"data": {"enterprise": {"organizations": {
			"nodes": [{"login": "org-a"}, null, {"login": "org-b"}, null],
			"pageInfo": {"hasNextPage": false, "endCursor": ""}
		}}},
		"errors": [
			{"message": "'kubearmor' forbids access via a personal access token (classic)."},
			{"message": "'openfga' forbids access via a personal access token (classic)."}
		]
	}`
	srv := newGraphQLServer(t, []string{resp})
	defer srv.Close()

	orgs, err := listEnterpriseOrgs(srv.Client(), srv.URL, "test-token", "cncf")
	if err != nil {
		t.Fatalf("listEnterpriseOrgs returned error: %v", err)
	}
	want := []string{"org-a", "org-b"}
	if !reflect.DeepEqual(orgs, want) {
		t.Errorf("orgs = %v, want %v", orgs, want)
	}
}

func TestListEnterpriseOrgsFailsWithoutData(t *testing.T) {
	resp := `{
		"data": {"enterprise": null},
		"errors": [{"message": "Bad credentials"}]
	}`
	srv := newGraphQLServer(t, []string{resp})
	defer srv.Close()

	_, err := listEnterpriseOrgs(srv.Client(), srv.URL, "test-token", "cncf")
	if err == nil {
		t.Fatal("expected error when response has errors and no data, got nil")
	}
	if !strings.Contains(err.Error(), "Bad credentials") {
		t.Errorf("error %q does not mention the GraphQL error message", err)
	}
}

func TestListEnterpriseOrgsPaginates(t *testing.T) {
	page1 := `{
		"data": {"enterprise": {"organizations": {
			"nodes": [{"login": "org-a"}],
			"pageInfo": {"hasNextPage": true, "endCursor": "cursor-1"}
		}}}
	}`
	page2 := `{
		"data": {"enterprise": {"organizations": {
			"nodes": [{"login": "org-b"}],
			"pageInfo": {"hasNextPage": false, "endCursor": ""}
		}}}
	}`
	srv := newGraphQLServer(t, []string{page1, page2})
	defer srv.Close()

	orgs, err := listEnterpriseOrgs(srv.Client(), srv.URL, "test-token", "cncf")
	if err != nil {
		t.Fatalf("listEnterpriseOrgs returned error: %v", err)
	}
	want := []string{"org-a", "org-b"}
	if !reflect.DeepEqual(orgs, want) {
		t.Errorf("orgs = %v, want %v", orgs, want)
	}
}
