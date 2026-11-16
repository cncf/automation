package main

import (
	"context"
	"errors"
	"log"
	"time"
)

// defaultRefreshInterval is how often the dataset is reloaded by default.
const defaultRefreshInterval = 6 * time.Hour

// Backoff for retrying a failed initial load. Variables so tests can lower them.
var (
	initialLoadRetryBaseDelay = 30 * time.Second
	initialLoadRetryMaxDelay  = 5 * time.Minute
)

func (s *serverState) hasDataset() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dataset != nil
}

// retryInitialLoad reloads the dataset with capped exponential backoff until
// one is loaded, so a source that was unreachable at startup does not leave
// the server empty until the next periodic refresh. It reports false when ctx
// is cancelled first.
func (s *serverState) retryInitialLoad(ctx context.Context) bool {
	delay := initialLoadRetryBaseDelay
	for !s.hasDataset() {
		log.Printf("no dataset loaded; retrying in %s", delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
		s.refresh(ctx)
		delay = min(delay*2, initialLoadRetryMaxDelay)
	}
	return true
}

// refresh reloads the dataset once. On success the new dataset replaces the
// old one under the write lock (in-flight requests keep their *Dataset
// snapshot). On failure or a 304 the previous dataset is left untouched.
func (s *serverState) refresh(ctx context.Context) {
	log.Println("dataset refresh started")
	start := time.Now()

	ds, validators, err := loadDataset(ctx, s.src, s.validators)
	switch {
	case errors.Is(err, errNotModified):
		log.Printf("dataset unchanged (checked in %s)", time.Since(start).Round(time.Millisecond))
	case err != nil:
		if s.hasDataset() {
			log.Printf("dataset refresh failed, keeping previous dataset: %v", err)
		} else {
			log.Printf("dataset load failed: %v", err)
		}
	default:
		s.mu.Lock()
		s.dataset = ds
		s.loadErr = nil
		s.mu.Unlock()
		s.validators = validators
		log.Printf("dataset refreshed (%d items) in %s", len(ds.Items), time.Since(start).Round(time.Millisecond))
	}
}

// runRefreshLoop first retries a failed initial load until a dataset is
// available, then refreshes it every interval until ctx is cancelled. A
// non-positive interval disables the periodic refresh but not the retries.
func (s *serverState) runRefreshLoop(ctx context.Context, interval time.Duration) {
	if !s.retryInitialLoad(ctx) || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.refresh(ctx)
		}
	}
}
