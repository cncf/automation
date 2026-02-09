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
		projectFile   = flag.String("project", "", "Path to project.yaml file (required)")
		thresholdDays = flag.Int("threshold", 180, "Days before considering maintainers stale")
		lastUpdate    = flag.String("last-update", "", "Override last update date (YYYY-MM-DD format)")
		outputFormat  = flag.String("output", "text", "Output format: text, json, yaml")
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

	var updateTime time.Time
	if *lastUpdate != "" {
		updateTime, err = time.Parse("2006-01-02", *lastUpdate)
		if err != nil {
			log.Fatalf("Invalid date format: %v", err)
		}
	} else {
		// Default: check file modification time
		info, err := os.Stat(*projectFile)
		if err != nil {
			log.Fatalf("Failed to stat file: %v", err)
		}
		updateTime = info.ModTime()
	}

	result := projects.CheckStaleness(project, updateTime, *thresholdDays)

	switch *outputFormat {
	case "json":
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	case "yaml":
		data, _ := yaml.Marshal(result)
		fmt.Print(string(data))
	default:
		results := []projects.StalenessResult{result}
		fmt.Print(projects.FormatStalenessResults(results))
	}

	if result.IsStale {
		os.Exit(1)
	}
}
