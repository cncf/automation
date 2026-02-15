package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type SBOMEntry struct {
	Project string `json:"project"`
	Repo    string `json:"repo"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type SubprojectSBOMEntry struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type Index struct {
	GeneratedAt     string                `json:"generated_at"`
	SBOMs           []SBOMEntry           `json:"sboms"`
	SubprojectSBOMs []SubprojectSBOMEntry `json:"subproject_sboms"`
}

func main() {
	baseDir := "."
	if len(os.Args) > 1 {
		baseDir = os.Args[1]
	}

	sbomDir := filepath.Join(baseDir, "supply-chain", "sbom")
	indexFile := filepath.Join(sbomDir, "index.json")

	var sboms []SBOMEntry
	var subprojectSBOMs []SubprojectSBOMEntry

	err := filepath.Walk(sbomDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".json") || strings.HasSuffix(path, "index.json") {
			return nil
		}

		// Get relative path from sbom directory
		relPath, err := filepath.Rel(sbomDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		parts := strings.Split(relPath, "/")

		// Check if it's a subproject SBOM
		if parts[0] == "subprojects" && len(parts) >= 5 {
			// subprojects/owner/repo/version/file.json
			subprojectSBOMs = append(subprojectSBOMs, SubprojectSBOMEntry{
				Owner:   parts[1],
				Repo:    parts[2],
				Version: parts[3],
				Path:    relPath,
			})
		} else if len(parts) >= 4 {
			// project/repo/version/file.json
			sboms = append(sboms, SBOMEntry{
				Project: parts[0],
				Repo:    parts[1],
				Version: parts[2],
				Path:    relPath,
			})
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	// Sort entries
	sort.Slice(sboms, func(i, j int) bool {
		if sboms[i].Project != sboms[j].Project {
			return sboms[i].Project < sboms[j].Project
		}
		if sboms[i].Repo != sboms[j].Repo {
			return sboms[i].Repo < sboms[j].Repo
		}
		return sboms[i].Version > sboms[j].Version // Newest first
	})

	sort.Slice(subprojectSBOMs, func(i, j int) bool {
		if subprojectSBOMs[i].Owner != subprojectSBOMs[j].Owner {
			return subprojectSBOMs[i].Owner < subprojectSBOMs[j].Owner
		}
		if subprojectSBOMs[i].Repo != subprojectSBOMs[j].Repo {
			return subprojectSBOMs[i].Repo < subprojectSBOMs[j].Repo
		}
		return subprojectSBOMs[i].Version > subprojectSBOMs[j].Version // Newest first
	})

	index := Index{
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		SBOMs:           sboms,
		SubprojectSBOMs: subprojectSBOMs,
	}

	// Write index file
	file, err := os.Create(indexFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating index file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(index); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated index with %d SBOMs and %d subproject SBOMs\n", len(sboms), len(subprojectSBOMs))
	fmt.Printf("Output: %s\n", indexFile)
}
