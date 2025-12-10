package projects

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaintainersConfig represents the maintainers configuration file
type MaintainersConfig struct {
	Maintainers []MaintainerEntry `yaml:"maintainers"`
}

// MaintainerEntry represents maintainers for a single project
type MaintainerEntry struct {
	ProjectID    string   `yaml:"project_id"`
	Org          string   `yaml:"org,omitempty"`
	Repository   string   `yaml:"repository,omitempty"`
	Branch       string   `yaml:"branch,omitempty"`
	Path         string   `yaml:"path,omitempty"`
	CanonicalURL string   `yaml:"canonical_url,omitempty"`
	Handles      []string `yaml:"handles"`
}

// MaintainerValidationResult captures validation results for maintainers
type MaintainerValidationResult struct {
	ProjectID             string   `json:"project_id" yaml:"project_id"`
	Org                   string   `json:"org,omitempty" yaml:"org,omitempty"`
	CanonicalURL          string   `json:"canonical_url,omitempty" yaml:"canonical_url,omitempty"`
	Valid                 bool     `json:"valid" yaml:"valid"`
	Errors                []string `json:"errors,omitempty" yaml:"errors,omitempty"`
	MissingHandles        []string `json:"missing_handles,omitempty" yaml:"missing_handles,omitempty"`
	ExtraHandles          []string `json:"extra_handles,omitempty" yaml:"extra_handles,omitempty"`
	LocalHandles          []string `json:"local_handles,omitempty" yaml:"local_handles,omitempty"`
	CanonicalHandles      []string `json:"canonical_handles,omitempty" yaml:"canonical_handles,omitempty"`
	VerificationAttempted bool     `json:"verification_attempted" yaml:"verification_attempted"`
	VerificationPassed    bool     `json:"verification_passed" yaml:"verification_passed"`
	VerifiedHandles       []string `json:"verified_handles,omitempty" yaml:"verified_handles,omitempty"`
}

// ValidateMaintainersFile validates a maintainers configuration file against canonical sources
func (pv *ProjectValidator) ValidateMaintainersFile(path string, verify bool) ([]MaintainerValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read maintainers file: %w", err)
	}

	var config MaintainersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse maintainers YAML: %w", err)
	}

	if len(config.Maintainers) == 0 {
		return nil, fmt.Errorf("maintainers file %s does not contain any entries", path)
	}

	var results []MaintainerValidationResult
	for _, entry := range config.Maintainers {
		result := pv.validateMaintainerEntry(entry, verify)
		results = append(results, result)
	}

	return results, nil
}

func (pv *ProjectValidator) validateMaintainerEntry(entry MaintainerEntry, verify bool) MaintainerValidationResult {
	result := MaintainerValidationResult{
		ProjectID: entry.ProjectID,
		Org:       entry.Org,
	}

	if entry.ProjectID == "" {
		result.Errors = append(result.Errors, "project_id is required")
	}

	cleanHandles, duplicateErrors := normalizeHandles(entry.Handles)
	result.LocalHandles = cleanHandles
	if len(cleanHandles) == 0 {
		result.Errors = append(result.Errors, "handles list cannot be empty")
	}
	result.Errors = append(result.Errors, duplicateErrors...)

	canonicalURL, err := resolveCanonicalURL(entry)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		result.CanonicalURL = canonicalURL
	}

	var canonicalHandles []string
	if result.CanonicalURL != "" {
		content, fetchErr := pv.fetchContent(result.CanonicalURL)
		if fetchErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch canonical maintainers: %v", fetchErr))
		} else {
			handles, parseErr := parseCanonicalMaintainers(content)
			if parseErr != nil {
				result.Errors = append(result.Errors, parseErr.Error())
			} else {
				canonicalHandles = handles
				result.CanonicalHandles = handles
			}
		}
	}

	missing, extra := compareHandles(cleanHandles, canonicalHandles)
	if len(missing) > 0 {
		result.MissingHandles = missing
	}
	if len(extra) > 0 {
		result.ExtraHandles = extra
	}

	if verify && len(cleanHandles) > 0 {
		result.VerificationAttempted = true
		var verified []string
		for _, handle := range cleanHandles {
			if err := pv.verifyHandleWithExternalService(entry.ProjectID, handle); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("verification failed for %s: %v", handle, err))
			} else {
				verified = append(verified, handle)
			}
		}
		if len(verified) == len(cleanHandles) && len(result.Errors) == 0 {
			result.VerificationPassed = true
		}
		result.VerifiedHandles = verified
	}

	result.Valid = len(result.Errors) == 0 && len(result.MissingHandles) == 0 && len(result.ExtraHandles) == 0
	if result.VerificationPassed && !result.Valid {
		result.VerificationPassed = false
	}
	return result
}

func normalizeHandles(handles []string) ([]string, []string) {
	seen := make(map[string]bool)
	var cleaned []string
	var errors []string
	for i, h := range handles {
		trimmed := strings.TrimSpace(h)
		if trimmed == "" {
			errors = append(errors, fmt.Sprintf("handles[%d] cannot be empty", i))
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, "@")
		key := strings.ToLower(trimmed)
		if seen[key] {
			errors = append(errors, fmt.Sprintf("duplicate handle detected: %s", trimmed))
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, trimmed)
	}
	sort.Strings(cleaned)
	return cleaned, errors
}

func resolveCanonicalURL(entry MaintainerEntry) (string, error) {
	if entry.CanonicalURL != "" {
		return os.ExpandEnv(entry.CanonicalURL), nil
	}
	if entry.Org == "" {
		return "", fmt.Errorf("org is required when canonical_url is not provided")
	}
	repo := entry.Repository
	if repo == "" {
		repo = ".project"
	}
	branch := entry.Branch
	if branch == "" {
		branch = "main"
	}
	path := entry.Path
	if path == "" {
		path = "MAINTAINERS.yaml"
	}

	return os.ExpandEnv(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", entry.Org, repo, branch, path)), nil
}

func parseCanonicalMaintainers(content string) ([]string, error) {
	type wrapper struct {
		Maintainers []string `yaml:"maintainers"`
		Handles     []string `yaml:"handles"`
	}

	var w wrapper
	if err := yaml.Unmarshal([]byte(content), &w); err == nil {
		if len(w.Maintainers) > 0 {
			return normalizeAndSort(w.Maintainers), nil
		}
		if len(w.Handles) > 0 {
			return normalizeAndSort(w.Handles), nil
		}
	}

	var list []string
	if err := yaml.Unmarshal([]byte(content), &list); err == nil && len(list) > 0 {
		return normalizeAndSort(list), nil
	}

	return nil, fmt.Errorf("canonical maintainers file is empty or in an unsupported format")
}

func normalizeAndSort(values []string) []string {
	normalized, _ := normalizeHandles(values)
	return normalized
}

func compareHandles(local []string, canonical []string) ([]string, []string) {
	localSet := make(map[string]struct{})
	for _, h := range local {
		localSet[strings.ToLower(h)] = struct{}{}
	}

	canonicalSet := make(map[string]struct{})
	for _, h := range canonical {
		canonicalSet[strings.ToLower(h)] = struct{}{}
	}

	var missing []string
	for _, h := range canonical {
		if _, ok := localSet[strings.ToLower(h)]; !ok {
			missing = append(missing, h)
		}
	}

	var extra []string
	for _, h := range local {
		if _, ok := canonicalSet[strings.ToLower(h)]; !ok {
			extra = append(extra, h)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

func (pv *ProjectValidator) verifyHandleWithExternalService(projectID, handle string) error {
	endpoint := os.Getenv("MAINTAINER_API_ENDPOINT")
	if endpoint == "" {
		log.Printf("[maintainers] Skipping external verification for %s (no MAINTAINER_API_ENDPOINT set)", handle)
		return nil
	}

	log.Printf("[maintainers] Stub verifying handle %s for project %s via %s", handle, projectID, endpoint)

	if strings.EqualFold(os.Getenv("MAINTAINER_API_STUB"), "fail") {
		return fmt.Errorf("stubbed failure for handle %s", handle)
	}

	return nil
}

// FormatMaintainersResults formats maintainer validation results
func (pv *ProjectValidator) FormatMaintainersResults(results []MaintainerValidationResult, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "yaml":
		data, err := yaml.Marshal(results)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return formatMaintainersText(results), nil
	}
}

func formatMaintainersText(results []MaintainerValidationResult) string {
	var b strings.Builder
	b.WriteString("Maintainers Validation Report\n")
	b.WriteString("============================\n\n")

	var invalidCount int
	for _, result := range results {
		if !result.Valid {
			invalidCount++
			b.WriteString(fmt.Sprintf("INVALID: %s", result.ProjectID))
			if result.CanonicalURL != "" {
				b.WriteString(fmt.Sprintf(" (canonical: %s)", result.CanonicalURL))
			}
			b.WriteString("\n")
			for _, err := range result.Errors {
				b.WriteString(fmt.Sprintf("  - %s\n", err))
			}
			if len(result.MissingHandles) > 0 {
				b.WriteString("  Missing handles:\n")
				for _, h := range result.MissingHandles {
					b.WriteString(fmt.Sprintf("    - %s\n", h))
				}
			}
			if len(result.ExtraHandles) > 0 {
				b.WriteString("  Extra handles:\n")
				for _, h := range result.ExtraHandles {
					b.WriteString(fmt.Sprintf("    - %s\n", h))
				}
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(fmt.Sprintf("Summary: %d maintainer entries validated, %d with issues\n", len(results), invalidCount))
	return b.String()
}
