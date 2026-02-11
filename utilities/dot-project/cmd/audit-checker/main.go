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
		projectFile  = flag.String("project", "", "Path to project.yaml file (required)")
		outputFormat = flag.String("output", "text", "Output format: text, json, yaml")
		timeout      = flag.Int("timeout", 10, "HTTP request timeout in seconds")
	)
	flag.Parse()

	if *projectFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -project flag is required")
		flag.Usage()
		os.Exit(1)
	}

	project, err := projects.LoadProjectFromFile(*projectFile)
	if err != nil {
		log.Fatalf("Failed to load project: %v", err)
	}

	client := &http.Client{Timeout: time.Duration(*timeout) * time.Second}
	result := projects.AuditProject(project, client)

	switch *outputFormat {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(result)
		fmt.Print(string(data))
	default:
		fmt.Print(projects.FormatAuditResult(result))
	}

	if result.FailCount > 0 {
		os.Exit(1)
	}
}
