package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"projects"

	"gopkg.in/yaml.v3"
)

func main() {
	var (
		projectFile  = flag.String("project", "", "Path to a single project.yaml file; defaults to auditing every project in the current directory")
		repoRoot     = flag.String("repo-root", "", "Root of a .project repository; audits every project it contains (default \".\")")
		outputFormat = flag.String("output", "text", "Output format: text, json, yaml")
		timeout      = flag.Int("timeout", 10, "HTTP request timeout in seconds")
	)
	flag.Parse()

	// With no explicit target, audit the repository in the working directory.
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

	client := &http.Client{Timeout: time.Duration(*timeout) * time.Second}

	var results []projects.AuditResult
	failed := false
	for _, path := range projectFiles {
		project, err := projects.LoadProjectFromFile(path)
		if err != nil {
			// One unreadable project must not hide its siblings' results,
			// which is the whole point of auditing a repository at once.
			fmt.Fprintf(os.Stderr, "Failed to load project %s: %v\n", path, err)
			failed = true
			continue
		}
		results = append(results, projects.AuditProject(project, client))
	}

	switch *outputFormat {
	case "json":
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(results)
		fmt.Print(string(data))
	default:
		for _, result := range results {
			fmt.Print(projects.FormatAuditResult(result))
		}
	}

	// Any failed URL fails the run, so one project's broken links cannot be
	// masked by a healthy sibling.
	for _, result := range results {
		if result.FailCount > 0 {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}
