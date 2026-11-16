package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// serverVersion is the semantic version of the MCP server, reported in the
// initialize handshake. It is a variable so release builds can override it with
// -ldflags "-X main.serverVersion=...".
var serverVersion = "0.2.0"

const (
	// maxLineBytes is the largest single JSON-RPC line accepted on stdin.
	maxLineBytes = 16 * 1024 * 1024
)

// datasetWaitTimeout bounds how long a request waits for the dataset to load.
var datasetWaitTimeout = 60 * time.Second

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
	// validators are the ETag/Last-Modified of the dataset currently served.
	// They are only touched by the goroutine that loads and refreshes the
	// dataset (initial load, then the refresh loop), so they need no lock.
	validators fetchValidators
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
	refreshInterval := flag.Duration("refresh-interval", defaultRefreshInterval, "How often to reload the dataset from its source (0 disables periodic refresh)")
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
		ds, validators, err := loadDataset(ctx, src, fetchValidators{})
		state.validators = validators
		state.setDataset(ds, err)
		if err != nil {
			log.Printf("error loading dataset: %v", err)
		} else {
			log.Printf("dataset loaded (%d items); server ready", len(ds.Items))
		}
		state.runRefreshLoop(ctx, *refreshInterval)
	}()

	reader := bufio.NewReaderSize(os.Stdin, 64*1024)
	for {
		raw, tooLong, err := readLine(reader, maxLineBytes)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("stdin read error: %v", err)
			}
			return
		}
		if tooLong {
			log.Printf("skipping stdin line longer than %d bytes", maxLineBytes)
			continue
		}

		line := strings.TrimSpace(string(raw))
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
}

// readLine reads one newline-terminated line of at most max bytes. A longer
// line is consumed in full and reported via tooLong so the caller can skip it.
// A final unterminated line at EOF is returned with a nil error; io.EOF is
// returned only once no more data remains.
func readLine(r *bufio.Reader, max int) (line []byte, tooLong bool, err error) {
	for {
		chunk, readErr := r.ReadSlice('\n')
		if !tooLong {
			if len(line)+len(chunk) > max {
				tooLong = true
				line = nil
			} else {
				line = append(line, chunk...)
			}
		}
		switch {
		case errors.Is(readErr, bufio.ErrBufferFull):
			continue
		case readErr == nil:
			return line, tooLong, nil
		case errors.Is(readErr, io.EOF) && (len(line) > 0 || tooLong):
			return line, tooLong, nil
		default:
			return nil, false, readErr
		}
	}
}
