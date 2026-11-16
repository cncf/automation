package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Fetch tuning. Variables (not constants) so tests can lower them.
var (
	// maxDatasetBytes caps the size of a remote dataset body.
	maxDatasetBytes int64 = 64 << 20
	// fetchAttempts is the number of tries for transient failures.
	fetchAttempts = 3
	// fetchRetryBaseDelay is the first backoff; it doubles per retry (1s, 2s).
	fetchRetryBaseDelay = time.Second
	// fetchTimeout bounds each individual HTTP attempt.
	fetchTimeout = 30 * time.Second
)

// errNotModified is returned (wrapped) when a conditional GET answers 304.
var errNotModified = errors.New("dataset not modified")

// fetchValidators are the HTTP cache validators of a fetched dataset.
type fetchValidators struct {
	etag         string
	lastModified string
}

// retryableError marks a failure worth another attempt.
type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func readSource(ctx context.Context, src dataSource, prev fetchValidators) ([]byte, fetchValidators, error) {
	switch {
	case src.filePath != "":
		raw, err := os.ReadFile(src.filePath)
		return raw, fetchValidators{}, err
	case src.url != "":
		return fetchURL(ctx, src.url, prev)
	default:
		return nil, fetchValidators{}, errors.New("either file or url must be provided")
	}
}

// fetchURL downloads url, retrying transient failures with exponential backoff.
func fetchURL(ctx context.Context, url string, prev fetchValidators) ([]byte, fetchValidators, error) {
	var lastErr error
	for attempt := range fetchAttempts {
		if attempt > 0 {
			delay := fetchRetryBaseDelay << (attempt - 1)
			log.Printf("dataset fetch failed (%v); retrying in %s", lastErr, delay)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, fetchValidators{}, ctx.Err()
			case <-timer.C:
			}
		}

		body, validators, err := fetchOnce(ctx, url, prev)
		var transient *retryableError
		if err == nil || !errors.As(err, &transient) || ctx.Err() != nil {
			return body, validators, err
		}
		lastErr = err
	}
	return nil, fetchValidators{}, fmt.Errorf("giving up after %d attempts: %w", fetchAttempts, lastErr)
}

func fetchOnce(ctx context.Context, url string, prev fetchValidators) ([]byte, fetchValidators, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fetchValidators{}, err
	}
	if prev.etag != "" {
		req.Header.Set("If-None-Match", prev.etag)
	}
	if prev.lastModified != "" {
		req.Header.Set("If-Modified-Since", prev.lastModified)
	}

	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fetchValidators{}, &retryableError{err}
	}
	defer func() { _ = resp.Body.Close() }()

	hasValidators := prev != fetchValidators{}
	switch {
	case resp.StatusCode == http.StatusNotModified && hasValidators:
		return nil, prev, errNotModified
	case resp.StatusCode >= http.StatusInternalServerError:
		return nil, fetchValidators{}, &retryableError{fmt.Errorf("unexpected status %d", resp.StatusCode)}
	case resp.StatusCode != http.StatusOK:
		return nil, fetchValidators{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDatasetBytes+1))
	if err != nil {
		return nil, fetchValidators{}, &retryableError{err}
	}
	if int64(len(body)) > maxDatasetBytes {
		return nil, fetchValidators{}, fmt.Errorf("response body exceeds %d bytes", maxDatasetBytes)
	}
	return body, fetchValidators{
		etag:         resp.Header.Get("ETag"),
		lastModified: resp.Header.Get("Last-Modified"),
	}, nil
}
