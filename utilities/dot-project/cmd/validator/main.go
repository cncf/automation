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
		configFile          = flag.String("config", "yaml/projectlist.yaml", "Path to project list configuration file")
		cacheDir            = flag.String("cache", ".cache", "Directory to store cached validation results")
		maintainersFile     = flag.String("maintainers", "yaml/maintainers.yaml", "Path to maintainers file (set empty to skip)")
		baseMaintainersFile = flag.String("base-maintainers", "", "Path to base maintainers file for diff validation")
		verifyMaintainers   = flag.Bool("verify-maintainers", false, "Verify maintainer handles via external service (stubbed)")
	)
	flag.Parse()

	if *configFile == "" {
		// Create a temporary dummy config file if none provided
		f, err := os.CreateTemp("", "dummy-projectlist-*.yaml")
		if err != nil {
			log.Fatal("failed to create temporary config file")
		}
		f.WriteString("projects: []")
		f.Close()
		defer os.Remove(f.Name())
		*configFile = f.Name()
	}

	validator := projects.NewValidator(*cacheDir)

	projectResults, err := validator.ValidateAll(*configFile)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	var maintainerResults []projects.MaintainerValidationResult
	maintainersEnabled := *maintainersFile != ""
	if maintainersEnabled {
		var excludedHandles map[string]bool
		if *baseMaintainersFile != "" {
			handles, err := validator.ExtractHandles(*baseMaintainersFile)
			if err != nil {
				log.Fatalf("failed to extract handles from base maintainers file: %v", err)
			}
			excludedHandles = handles
		}

		results, err := validator.ValidateMaintainersFileWithExclusion(*maintainersFile, *verifyMaintainers, excludedHandles)
		if err != nil {
			log.Fatalf("maintainers validation failed: %v", err)
		}
		maintainerResults = results
	}

	output, err := validator.FormatResults(projectResults, "text")
	if err != nil {
		log.Fatalf("failed to format project results: %v", err)
	}
	fmt.Print(output)
	if maintainersEnabled {
		fmt.Println()
		maintainersOutput, err := validator.FormatMaintainersResults(maintainerResults, "text")
		if err != nil {
			log.Fatalf("failed to format maintainer results: %v", err)
		}
		fmt.Print(maintainersOutput)
	}

	// Check if any validation failed
	hasErrors := false
	for _, result := range projectResults {
		if !result.Valid {
			hasErrors = true
			break
		}
	}
	if !hasErrors && maintainersEnabled {
		for _, result := range maintainerResults {
			if !result.Valid {
				hasErrors = true
				break
			}
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}
