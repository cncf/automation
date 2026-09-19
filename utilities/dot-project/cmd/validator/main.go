package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"projects"
)

func main() {
	var (
		configFile          = flag.String("config", "yaml/projectlist.yaml", "Path to project list configuration file")
		repoRoot            = flag.String("repo-root", "", "Root of a .project repository; discovers and validates every project it contains (overrides -config and -maintainers)")
		cacheDir            = flag.String("cache", ".cache", "Directory to store cached validation results")
		maintainersFile     = flag.String("maintainers", "yaml/maintainers.yaml", "Path to maintainers file (set empty to skip)")
		baseMaintainersFile = flag.String("base-maintainers", "", "Path to base maintainers file, or to a base repository root, for diff validation")
		verifyMaintainers   = flag.Bool("verify-maintainers", false, "Verify maintainer handles via external service (stubbed)")
		outputFormat        = flag.String("output", "text", "Output format: text, json, yaml")
	)
	flag.Parse()

	// The composite actions each cover one half of the repository, so
	// repo-root mode has to honour the established "skip" idioms: an explicit
	// -maintainers "" means projects only, and an explicit -config /dev/null
	// means maintainers only. Without this, running both actions on a
	// multi-project repo would report every file twice.
	skipProjects, skipMaintainers := false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "config":
			v := f.Value.String()
			skipProjects = v == "" || v == "/dev/null"
		case "maintainers":
			skipMaintainers = f.Value.String() == ""
		}
	})

	// In repo-root mode the layout decides which files to validate, so the
	// explicit -config and -maintainers paths are replaced by discovery.
	var discovery *projects.Discovery
	if *repoRoot != "" {
		d, err := projects.Discover(*repoRoot)
		if err != nil {
			log.Fatalf("discovery failed: %v", err)
		}
		discovery = d

		*configFile = ""
		if !skipProjects {
			listFile, cleanup, err := writeProjectList(d)
			if err != nil {
				log.Fatalf("failed to build project list: %v", err)
			}
			defer cleanup()
			*configFile = listFile
		}

		*maintainersFile = ""
	}

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

	// Repository-wide checks run first: when the layout itself is wrong, the
	// per-file results are reported against files that may not be the ones
	// the repository actually intends to ship.
	repoValid := true
	if discovery != nil {
		repoResult, err := projects.ValidateRepo(*repoRoot)
		if err != nil {
			log.Fatalf("repository validation failed: %v", err)
		}
		repoOutput, err := projects.FormatRepoResult(repoResult, *outputFormat)
		if err != nil {
			log.Fatalf("failed to format repository results: %v", err)
		}
		fmt.Print(repoOutput)
		fmt.Println()
		repoValid = repoResult.Valid
	}

	projectResults, err := validator.ValidateAll(*configFile)
	if err != nil {
		log.Fatalf("validation failed: %v", err)
	}

	var excludedHandles map[string]bool
	if *baseMaintainersFile != "" {
		handles, err := validator.ExtractHandlesFrom(*baseMaintainersFile)
		if err != nil {
			log.Fatalf("failed to extract handles from base maintainers path: %v", err)
		}
		excludedHandles = handles
	}

	var maintainerResults []projects.MaintainerValidationResult
	maintainersEnabled := *maintainersFile != ""
	if maintainersEnabled {
		results, err := validator.ValidateMaintainersFileWithExclusion(*maintainersFile, *verifyMaintainers, excludedHandles)
		if err != nil {
			log.Fatalf("maintainers validation failed: %v", err)
		}
		maintainerResults = results
	}

	if discovery != nil && !skipMaintainers {
		for _, path := range discovery.MaintainersPaths() {
			results, err := validator.ValidateMaintainersFileWithExclusion(path, *verifyMaintainers, excludedHandles)
			if err != nil {
				// ValidateRepo already reported this file as unreadable and
				// cleared repoValid. Aborting here would hide the sibling
				// projects that are fine, which is the opposite of what a
				// multi-project report is for.
				continue
			}
			maintainerResults = append(maintainerResults, results...)
			maintainersEnabled = true
		}
	}

	output, err := validator.FormatResults(projectResults, *outputFormat)
	if err != nil {
		log.Fatalf("failed to format project results: %v", err)
	}
	fmt.Print(output)
	if maintainersEnabled {
		fmt.Println()
		maintainersOutput, err := validator.FormatMaintainersResults(maintainerResults, *outputFormat)
		if err != nil {
			log.Fatalf("failed to format maintainer results: %v", err)
		}
		fmt.Print(maintainersOutput)
	}

	// Check if any validation failed
	hasErrors := !repoValid
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

// writeProjectList materializes a discovery result as a temporary project list
// file, so repo-root mode reuses the same fetch, cache and diff pipeline as an
// explicit -config run.
func writeProjectList(d *projects.Discovery) (string, func(), error) {
	f, err := os.CreateTemp("", "discovered-projectlist-*.yaml")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { os.Remove(f.Name()) }

	var b strings.Builder
	b.WriteString("projects:\n")
	for _, p := range d.Projects {
		abs, err := filepath.Abs(p.ProjectPath)
		if err != nil {
			f.Close()
			cleanup()
			return "", func() {}, err
		}

		// A declared project whose file is absent is already reported by
		// ValidateRepo, which names the exact expected path. Listing it here
		// too would restate the same problem as an opaque fetch failure.
		if _, err := os.Stat(abs); err != nil {
			continue
		}

		id := p.ID
		if id == "" {
			// Single-project repos have no declared id; the slug is not known
			// until the file is parsed, so fall back to the directory name.
			id = filepath.Base(filepath.Dir(abs))
		}

		b.WriteString(fmt.Sprintf("  - url: %q\n", "file://"+filepath.ToSlash(abs)))
		b.WriteString(fmt.Sprintf("    id: %q\n", id))
	}

	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return f.Name(), cleanup, nil
}
