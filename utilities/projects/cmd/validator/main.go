package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"projects"
)

func main() {
	var (
		configFile = flag.String("config", "yaml/projectlist.yaml", "Path to project list configuration file")
		cacheDir   = flag.String("cache", ".cache", "Directory to store cached validation results")
		format     = flag.String("format", "text", "Output format: text, json")
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("config file is required")
	}

	validator := projects.NewValidator(*cacheDir)

	results, err := validator.ValidateAll(*configFile)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	switch *format {
	case "json":
		output, err := validator.FormatResults(results, "json")
		if err != nil {
			log.Fatalf("failed to format results: %v", err)
		}
		fmt.Print(output)
	default:
		output, err := validator.FormatResults(results, "text")
		if err != nil {
			log.Fatalf("failed to format results: %v", err)
		}
		fmt.Print(output)
	}

	// Check if any validation failed
	hasErrors := false
	for _, result := range results {
		if !result.Valid {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}
