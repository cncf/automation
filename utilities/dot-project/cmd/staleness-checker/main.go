package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"projects"

	"gopkg.in/yaml.v3"
)

func main() {
	var (
		projectFile   = flag.String("project", "", "Path to a single project.yaml file; defaults to checking every project in the current directory")
		repoRoot      = flag.String("repo-root", "", "Root of a .project repository; checks every project it contains (default \".\")")
		thresholdDays = flag.Int("threshold", projects.DefaultStalenessThresholdDays, "Days before considering maintainers stale")
		lastUpdate    = flag.String("last-update", "", "Override last update date (YYYY-MM-DD format)")
		outputFormat  = flag.String("output", "text", "Output format: text, json, yaml")
	)
	flag.Parse()

	// With no explicit target, check the repository in the working directory.
	// Discover reads the layout from disk, so this works unchanged for both
	// legacy single-project repos and multi-project ones.
	switch {
	case *projectFile == "" && *repoRoot == "":
		*repoRoot = "."
	case *projectFile != "" && *repoRoot != "":
		fmt.Fprintln(os.Stderr, "Error: -project and -repo-root are mutually exclusive")
		flag.Usage()
		os.Exit(1)
	}

	projectFiles := []string{*projectFile}
	if *repoRoot != "" {
		d, err := projects.Discover(*repoRoot)
		if err != nil {
			log.Fatalf("Discovery failed: %v", err)
		}
		projectFiles = d.ProjectPaths()
	}

	var override time.Time
	if *lastUpdate != "" {
		parsed, err := time.Parse("2006-01-02", *lastUpdate)
		if err != nil {
			log.Fatalf("Invalid date format: %v", err)
		}
		override = parsed
	}

	var results []projects.StalenessResult
	failed := false
	for _, path := range projectFiles {
		result, err := checkProject(path, override, *thresholdDays)
		if err != nil {
			// One unreadable project must not hide its siblings' results,
			// which is the whole point of checking a repository at once.
			fmt.Fprintf(os.Stderr, "%v\n", err)
			failed = true
			continue
		}
		results = append(results, result)
	}

	switch *outputFormat {
	case "json":
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(results)
		fmt.Print(string(data))
	default:
		fmt.Print(projects.FormatStalenessResults(results))
	}

	// Any stale project fails the run, so a multi-project repository cannot
	// hide one stale roster behind a fresh sibling.
	for _, result := range results {
		if result.IsStale {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func checkProject(path string, override time.Time, thresholdDays int) (projects.StalenessResult, error) {
	project, err := projects.LoadProjectFromFile(path)
	if err != nil {
		return projects.StalenessResult{}, fmt.Errorf("failed to load project %s: %w", path, err)
	}

	updateTime := override
	if updateTime.IsZero() {
		// Default: check file modification time
		info, err := os.Stat(path)
		if err != nil {
			return projects.StalenessResult{}, fmt.Errorf("failed to stat %s: %w", path, err)
		}
		updateTime = info.ModTime()
	}

	return projects.CheckStaleness(project, updateTime, thresholdDays), nil
}
