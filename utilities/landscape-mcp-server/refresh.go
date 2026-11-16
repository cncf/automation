package main

import (
	"context"
	"errors"
	"log"
	"time"
)

// defaultRefreshInterval is how often the dataset is reloaded by default.
const defaultRefreshInterval = 6 * time.Hour

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
		log.Printf("dataset refresh failed, keeping previous dataset: %v", err)
	default:
		s.mu.Lock()
		s.dataset = ds
		s.loadErr = nil
		s.mu.Unlock()
		s.validators = validators
		log.Printf("dataset refreshed (%d items) in %s", len(ds.Items), time.Since(start).Round(time.Millisecond))
	}
}

// runRefreshLoop refreshes the dataset every interval until ctx is cancelled.
// A non-positive interval disables periodic refresh.
func (s *serverState) runRefreshLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
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
