package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const testETag = `"v1"`

func sampleBody(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile("testdata/sample.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return body
}

// fastRetries removes the real backoff delay for the duration of the test.
func fastRetries(t *testing.T) {
	t.Helper()
	orig := fetchRetryBaseDelay
	fetchRetryBaseDelay = time.Millisecond
	t.Cleanup(func() { fetchRetryBaseDelay = orig })
}

func TestLoadDataset_loads_remote_body_and_captures_validators(t *testing.T) {
	// Given a server returning the fixture with cache validators
	body := sampleBody(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", testETag)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// When loading without cached validators
	ds, v, err := loadDataset(context.Background(), dataSource{url: srv.URL}, fetchValidators{})

	// Then the dataset parses and the validators are returned
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ds.Items) != 6 {
		t.Errorf("expected 6 items, got %d", len(ds.Items))
	}
	want := fetchValidators{etag: testETag, lastModified: "Mon, 02 Jan 2006 15:04:05 GMT"}
	if v != want {
		t.Errorf("validators = %+v, want %+v", v, want)
	}
}

func TestLoadDataset_reports_not_modified_when_validators_match(t *testing.T) {
	// Given a server that answers 304 only for matching conditional headers
	var gotINM, gotIMS atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotINM.Store(r.Header.Get("If-None-Match"))
		gotIMS.Store(r.Header.Get("If-Modified-Since"))
		w.WriteHeader(http.StatusNotModified)
	}))
	defer srv.Close()
	prev := fetchValidators{etag: testETag, lastModified: "Mon, 02 Jan 2006 15:04:05 GMT"}

	// When loading with cached validators
	ds, _, err := loadDataset(context.Background(), dataSource{url: srv.URL}, prev)

	// Then not-modified is signalled and the validators were sent
	if !errors.Is(err, errNotModified) {
		t.Fatalf("err = %v, want errNotModified", err)
	}
	if ds != nil {
		t.Error("expected nil dataset on 304")
	}
	if gotINM.Load() != prev.etag || gotIMS.Load() != prev.lastModified {
		t.Errorf("conditional headers = %v / %v", gotINM.Load(), gotIMS.Load())
	}
}

func TestLoadDataset_initial_load_sends_no_validators(t *testing.T) {
	// Given a server recording the conditional headers it receives
	body := sampleBody(t)
	var conditional atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Modified-Since") != "" {
			conditional.Store(true)
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// When doing the initial load
	if _, _, err := loadDataset(context.Background(), dataSource{url: srv.URL}, fetchValidators{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then the request was unconditional
	if conditional.Load() {
		t.Error("initial load must not send conditional headers")
	}
}

func TestFetchURL_retries_5xx_then_succeeds(t *testing.T) {
	// Given a server failing twice with 503 before succeeding
	fastRetries(t)
	body := sampleBody(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// When fetching
	got, _, err := fetchURL(context.Background(), srv.URL, fetchValidators{})

	// Then the third attempt succeeds
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Error("body mismatch")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestFetchURL_gives_up_after_max_attempts(t *testing.T) {
	// Given a server that always fails with 500
	fastRetries(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// When fetching
	_, _, err := fetchURL(context.Background(), srv.URL, fetchValidators{})

	// Then it errors after exactly fetchAttempts tries
	if err == nil {
		t.Fatal("expected error")
	}
	if int(calls.Load()) != fetchAttempts {
		t.Errorf("calls = %d, want %d", calls.Load(), fetchAttempts)
	}
}

func TestFetchURL_does_not_retry_4xx(t *testing.T) {
	// Given a server answering 404
	fastRetries(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// When fetching
	_, _, err := fetchURL(context.Background(), srv.URL, fetchValidators{})

	// Then a single attempt is made
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestFetchURL_stops_retrying_when_context_cancelled(t *testing.T) {
	// Given a failing server and a long backoff
	orig := fetchRetryBaseDelay
	fetchRetryBaseDelay = time.Minute
	t.Cleanup(func() { fetchRetryBaseDelay = orig })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// When fetching
	start := time.Now()
	_, _, err := fetchURL(ctx, srv.URL, fetchValidators{})

	// Then the backoff wait is aborted by the context
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want deadline exceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Error("backoff did not respect context")
	}
}

func TestFetchURL_rejects_oversized_body_without_retry(t *testing.T) {
	// Given a body larger than the lowered cap
	orig := maxDatasetBytes
	maxDatasetBytes = 16
	t.Cleanup(func() { maxDatasetBytes = orig })
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(strings.Repeat("x", 17)))
	}))
	defer srv.Close()

	// When fetching
	_, _, err := fetchURL(context.Background(), srv.URL, fetchValidators{})

	// Then it fails once, without retrying
	if err == nil {
		t.Fatal("expected error for oversized body")
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}

func TestFetchURL_accepts_body_exactly_at_cap(t *testing.T) {
	// Given a body of exactly the cap size
	orig := maxDatasetBytes
	maxDatasetBytes = 16
	t.Cleanup(func() { maxDatasetBytes = orig })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 16)))
	}))
	defer srv.Close()

	// When fetching
	got, _, err := fetchURL(context.Background(), srv.URL, fetchValidators{})

	// Then it succeeds
	if err != nil || len(got) != 16 {
		t.Errorf("got %d bytes, err %v", len(got), err)
	}
}

// refreshFixture serves the fixture while healthy and 500 when failing is set.
type refreshFixture struct {
	srv     *httptest.Server
	failing atomic.Bool
	calls   atomic.Int32
	notMod  atomic.Bool
}

func newRefreshFixture(t *testing.T) *refreshFixture {
	t.Helper()
	body := sampleBody(t)
	f := &refreshFixture{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		switch {
		case f.failing.Load():
			w.WriteHeader(http.StatusInternalServerError)
		case f.notMod.Load() && r.Header.Get("If-None-Match") == testETag:
			w.WriteHeader(http.StatusNotModified)
		default:
			w.Header().Set("ETag", testETag)
			_, _ = w.Write(body)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// loadedState returns a serverState that has completed its initial load.
func loadedState(t *testing.T, f *refreshFixture) *serverState {
	t.Helper()
	state := newServerState(LandscapeConfig{Name: "CNCF"})
	state.src = dataSource{url: f.srv.URL}
	ds, v, err := loadDataset(context.Background(), state.src, fetchValidators{})
	if err != nil {
		t.Fatalf("initial load: %v", err)
	}
	state.validators = v
	state.setDataset(ds, nil)
	return state
}

func TestRefresh_failure_keeps_previous_dataset(t *testing.T) {
	// Given a loaded state whose source then starts failing
	fastRetries(t)
	f := newRefreshFixture(t)
	state := loadedState(t, f)
	before := state.dataset
	f.failing.Store(true)

	// When refreshing
	state.refresh(context.Background())

	// Then the previous dataset is still served, with no error recorded
	got, err := state.waitForDataset(context.Background())
	if err != nil {
		t.Fatalf("waitForDataset: %v", err)
	}
	if got != before {
		t.Error("failed refresh replaced the dataset")
	}
}

func TestRefresh_not_modified_keeps_dataset(t *testing.T) {
	// Given a loaded state whose source answers 304 to its ETag
	f := newRefreshFixture(t)
	state := loadedState(t, f)
	before := state.dataset
	f.notMod.Store(true)

	// When refreshing
	state.refresh(context.Background())

	// Then the same dataset pointer is retained and the validators persist
	if state.dataset != before {
		t.Error("304 replaced the dataset")
	}
	if state.validators.etag != testETag {
		t.Errorf("validators lost: %+v", state.validators)
	}
}

func TestRefresh_success_swaps_dataset(t *testing.T) {
	// Given a loaded state whose source returns a fresh 200
	f := newRefreshFixture(t)
	state := loadedState(t, f)
	before := state.dataset

	// When refreshing
	state.refresh(context.Background())

	// Then a new dataset is installed; the old snapshot is untouched
	if state.dataset == before {
		t.Error("expected new dataset after successful refresh")
	}
	if len(before.Items) != 6 || len(state.dataset.Items) != 6 {
		t.Error("dataset contents changed unexpectedly")
	}
}

func TestRefresh_recovers_after_failed_initial_load(t *testing.T) {
	// Given a state whose initial load failed
	f := newRefreshFixture(t)
	state := newServerState(LandscapeConfig{Name: "CNCF"})
	state.src = dataSource{url: f.srv.URL}
	state.setDataset(nil, errors.New("boom"))

	// When a refresh succeeds
	state.refresh(context.Background())

	// Then requests are served again
	if _, err := state.waitForDataset(context.Background()); err != nil {
		t.Errorf("waitForDataset after recovery: %v", err)
	}
}

func TestRunRefreshLoop_refreshes_periodically_and_stops_on_cancel(t *testing.T) {
	// Given a state refreshed on a tiny interval
	f := newRefreshFixture(t)
	state := loadedState(t, f)
	initial := f.calls.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})

	// When the loop runs until at least two refreshes happened
	go func() {
		state.runRefreshLoop(ctx, 5*time.Millisecond)
		close(done)
	}()
	deadline := time.After(time.Second)
	for f.calls.Load() < initial+2 {
		select {
		case <-deadline:
			t.Fatal("loop did not refresh in time")
		case <-time.After(time.Millisecond):
		}
	}

	// Then cancelling the context stops it
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loop did not stop after cancel")
	}
}

// fastInitialLoadRetries shrinks the initial-load backoff for the test.
func fastInitialLoadRetries(t *testing.T) {
	t.Helper()
	base, max := initialLoadRetryBaseDelay, initialLoadRetryMaxDelay
	initialLoadRetryBaseDelay, initialLoadRetryMaxDelay = time.Millisecond, 4*time.Millisecond
	t.Cleanup(func() { initialLoadRetryBaseDelay, initialLoadRetryMaxDelay = base, max })
}

// failedState returns a serverState whose initial load failed.
func failedState(f *refreshFixture) *serverState {
	state := newServerState(LandscapeConfig{Name: "CNCF"})
	state.src = dataSource{url: f.srv.URL}
	state.setDataset(nil, errors.New("boom"))
	return state
}

func TestRunRefreshLoop_retries_failed_initial_load_even_when_refresh_disabled(t *testing.T) {
	// Given a state whose initial load failed and a source that is still down
	fastRetries(t)
	fastInitialLoadRetries(t)
	f := newRefreshFixture(t)
	f.failing.Store(true)
	state := failedState(f)
	done := make(chan struct{})

	// When the loop runs with periodic refresh disabled
	go func() {
		state.runRefreshLoop(context.Background(), 0)
		close(done)
	}()

	// Then it keeps retrying while the source is down
	deadline := time.After(2 * time.Second)
	for int(f.calls.Load()) < 2*fetchAttempts {
		select {
		case <-done:
			t.Fatal("loop gave up while no dataset was loaded")
		case <-deadline:
			t.Fatal("loop did not retry the initial load")
		case <-time.After(time.Millisecond):
		}
	}

	// And once the source recovers the dataset is served and the loop ends
	f.failing.Store(false)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not finish after the dataset loaded")
	}
	if _, err := state.waitForDataset(context.Background()); err != nil {
		t.Errorf("waitForDataset after recovery: %v", err)
	}
}

func TestRunRefreshLoop_initial_load_retry_stops_on_cancel(t *testing.T) {
	// Given a state whose initial load failed and a source that stays down
	fastRetries(t)
	fastInitialLoadRetries(t)
	f := newRefreshFixture(t)
	f.failing.Store(true)
	state := failedState(f)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		state.runRefreshLoop(ctx, time.Hour)
		close(done)
	}()

	// When the context is cancelled
	cancel()

	// Then the loop stops without a dataset
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop after cancel")
	}
	if state.hasDataset() {
		t.Error("dataset loaded from a failing source")
	}
}

func TestRunRefreshLoop_zero_interval_returns_immediately(t *testing.T) {
	// Given a state and a disabled interval
	f := newRefreshFixture(t)
	state := loadedState(t, f)
	initial := f.calls.Load()

	// When running the loop
	state.runRefreshLoop(context.Background(), 0)

	// Then it returns without fetching
	if f.calls.Load() != initial {
		t.Error("disabled loop fetched")
	}
}

func TestLoadDataset_summarises_invalid_dates_in_one_log_line(t *testing.T) {
	// Given a dataset with several invalid dates across items
	path := t.TempDir() + "/bad.json"
	body := `{"items":[
		{"name":"A","accepted_at":"nope","joined_at":"nope"},
		{"name":"B","graduated_at":"nope","audits":[{"date":"nope"}]}
	]}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	// When loading
	ds, _, err := loadDataset(context.Background(), dataSource{filePath: path}, fetchValidators{})

	// Then loading succeeds and exactly one warning line is emitted
	if err != nil || len(ds.Items) != 2 {
		t.Fatalf("load: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "4 invalid dates") || !strings.Contains(lines[0], "A accepted_at") {
		t.Errorf("log output = %q", buf.String())
	}
}

func TestLoadDataset_rejects_malformed_json_from_file(t *testing.T) {
	// Given a file that is not valid JSON
	path := t.TempDir() + "/broken.json"
	if err := os.WriteFile(path, []byte(`{"items": [`), 0o600); err != nil {
		t.Fatal(err)
	}

	// When loading
	ds, _, err := loadDataset(context.Background(), dataSource{filePath: path}, fetchValidators{})

	// Then loading fails and no dataset is returned
	if err == nil || ds != nil {
		t.Fatalf("ds = %v, err = %v; want nil dataset and error", ds, err)
	}
}

func TestLoadDataset_rejects_malformed_json_from_url(t *testing.T) {
	// Given a server answering 200 with a non-JSON body
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer srv.Close()

	// When loading
	ds, _, err := loadDataset(context.Background(), dataSource{url: srv.URL}, fetchValidators{})

	// Then loading fails and no dataset is returned
	if err == nil || ds != nil {
		t.Fatalf("ds = %v, err = %v; want nil dataset and error", ds, err)
	}
}

func TestLoadDataset_propagates_non_retried_4xx(t *testing.T) {
	// Given a server answering 403
	fastRetries(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	// When loading
	ds, _, err := loadDataset(context.Background(), dataSource{url: srv.URL}, fetchValidators{})

	// Then loading fails after a single attempt
	if err == nil || ds != nil {
		t.Fatalf("ds = %v, err = %v; want nil dataset and error", ds, err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
}
