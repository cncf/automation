package projects

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateMaintainersFile validates a maintainers configuration file against canonical sources
func (pv *ProjectValidator) ValidateMaintainersFile(path string, verify bool) ([]MaintainerValidationResult, error) {
	return pv.ValidateMaintainersFileWithExclusion(path, verify, nil)
}

// ValidateMaintainersFileWithExclusion validates a maintainers configuration file, optionally excluding some handles from verification
func (pv *ProjectValidator) ValidateMaintainersFileWithExclusion(path string, verify bool, excludedHandles map[string]bool) ([]MaintainerValidationResult, error) {
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
		result := pv.validateMaintainerEntry(entry, verify, excludedHandles)
		results = append(results, result)
	}

	return results, nil
}

// ExtractHandles reads a maintainers file and returns a set of all handles
func (pv *ProjectValidator) ExtractHandles(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read maintainers file: %w", err)
	}

	var config MaintainersConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse maintainers YAML: %w", err)
	}

	handles := make(map[string]bool)
	for _, entry := range config.Maintainers {
		for _, team := range entry.Teams {
			for _, member := range team.Members {
				trimmed := strings.TrimSpace(member)
				trimmed = strings.TrimPrefix(trimmed, "@")
				if trimmed != "" {
					handles[strings.ToLower(trimmed)] = true
				}
			}
		}
	}
	return handles, nil
}

func (pv *ProjectValidator) validateMaintainerEntry(entry MaintainerEntry, verify bool, excludedHandles map[string]bool) MaintainerValidationResult {
	result := MaintainerValidationResult{
		ProjectID: entry.ProjectID,
		Org:       entry.Org,
	}

	if entry.ProjectID == "" {
		result.Errors = append(result.Errors, "project_id is required")
	}

	if len(entry.Teams) == 0 {
		result.Errors = append(result.Errors, "teams list cannot be empty")
	}

	hasProjectMaintainers := false
	var allVerifiedHandles []string
	allPassed := true

	for _, team := range entry.Teams {
		if team.Name == "project-maintainers" {
			hasProjectMaintainers = true
			if len(team.Members) == 0 {
				result.Errors = append(result.Errors, "team 'project-maintainers' cannot be empty")
			}
		}

		cleanHandles, duplicateErrors := normalizeHandles(team.Members)
		if len(duplicateErrors) > 0 {
			for _, err := range duplicateErrors {
				result.Errors = append(result.Errors, fmt.Sprintf("team '%s': %s", team.Name, err))
			}
		}

		if verify && len(cleanHandles) > 0 {
			result.VerificationAttempted = true
			for _, handle := range cleanHandles {
				if excludedHandles != nil && excludedHandles[strings.ToLower(handle)] {
					continue
				}
				if err := pv.verifyHandleWithExternalService(entry.ProjectID, handle); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("verification failed for %s (team %s): %v", handle, team.Name, err))
					allPassed = false
				} else {
					allVerifiedHandles = append(allVerifiedHandles, handle)
				}
			}
		}
	}

	if !hasProjectMaintainers {
		result.Errors = append(result.Errors, "team 'project-maintainers' is required")
	}

	if result.VerificationAttempted {
		result.VerificationPassed = allPassed && len(result.Errors) == 0
		result.VerifiedHandles = allVerifiedHandles
	}

	result.Valid = len(result.Errors) == 0
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

func (pv *ProjectValidator) verifyHandleWithExternalService(projectID, handle string) error {
	// If LFX_AUTH_TOKEN is set, perform LFX validation
	if os.Getenv("LFX_AUTH_TOKEN") != "" {
		if !checkMaintainerInLFX(handle) {
			return fmt.Errorf("handle %s not found in LFX", handle)
		}
		return nil
	}

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
			b.WriteString("\n")
			for _, err := range result.Errors {
				b.WriteString(fmt.Sprintf("  - %s\n", err))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(fmt.Sprintf("Summary: %d maintainer entries validated, %d with issues\n", len(results), invalidCount))
	return b.String()
}

func checkMaintainerInLFX(handle string) bool {
	token := os.Getenv("LFX_AUTH_TOKEN")
	if token == "" {
		log.Printf("LFX_AUTH_TOKEN environment variable is not set")
		return false
	}
	apiURL := "https://api-gw.platform.linuxfoundation.org/user-service/v1/users/search"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return false
	}

	q := req.URL.Query()
	q.Add("githubID", handle)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request to LFX: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("LFX API returned status: %d", resp.StatusCode)
		return false
	}

	var result struct {
		Data []interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response: %v", err)
		return false
	}

	return len(result.Data) > 0
}
