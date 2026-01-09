package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// NewProjectValidator creates a new project validator
func NewProjectValidator(configPath string) (*ProjectValidator, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	cache, err := loadCache(config.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load cache: %v", err)
	}

	return &ProjectValidator{
		config: config,
		cache:  cache,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// ValidateProjects validates all projects in the project list
func (pv *ProjectValidator) ValidateProjects() ([]ValidationResult, error) {
	// Load project list
	projectURLs, err := pv.loadProjectList()
	if err != nil {
		return nil, fmt.Errorf("failed to load project list: %v", err)
	}

	var results []ValidationResult
	for _, url := range projectURLs {
		result, err := pv.validateProject(url)
		if err != nil {
			log.Printf("Error validating project %s: %v", url, err)
			result = ValidationResult{
				URL:         url,
				Valid:       false,
				Errors:      []string{err.Error()},
				LastChecked: time.Now(),
			}
		}
		results = append(results, result)
	}

	// Save cache
	if err := pv.cache.save(); err != nil {
		log.Printf("Failed to save cache: %v", err)
	}

	return results, nil
}

// validateProject validates a single project YAML file
func (pv *ProjectValidator) validateProject(url string) (ValidationResult, error) {
	result := ValidationResult{
		URL:         url,
		LastChecked: time.Now(),
	}

	// Fetch content
	content, err := pv.fetchContent(url)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to fetch content: %v", err))
		return result, nil
	}

	// Calculate hash
	hash := calculateHash(content)
	result.CurrentHash = hash

	// Check if changed
	if cached, exists := pv.cache.Entries[url]; exists {
		result.PreviousHash = cached.Hash
		result.Changed = cached.Hash != hash
	} else {
		result.Changed = true // New project
	}

	// Parse and validate YAML
	var project Project
	if err := yaml.Unmarshal([]byte(content), &project); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("YAML parsing error: %v", err))
		result.Valid = false
	} else {
		result.ProjectName = project.Name
		// Validate project structure
		validationErrors := validateProjectStruct(project)
		if len(validationErrors) > 0 {
			result.Errors = append(result.Errors, validationErrors...)
			result.Valid = false
		} else {
			result.Valid = true
		}
	}

	// Update cache
	pv.cache.Entries[url] = CacheEntry{
		URL:         url,
		Hash:        hash,
		LastChecked: time.Now(),
		Content:     content,
	}

	return result, nil
}

// ProjectListEntry represents a single entry in the project list
type ProjectListEntry struct {
	URL string `yaml:"url"`
	ID  string `yaml:"id,omitempty"`
}

// ProjectListConfig represents the structure of the project list file
type ProjectListConfig struct {
	Projects []ProjectListEntry `yaml:"projects"`
}

// loadProjectList loads the list of project URLs
func (pv *ProjectValidator) loadProjectList() ([]string, error) {
	// For compatibility, check if projectListURL is set, otherwise use a default projectlist.yaml
	var projectListURL string
	if pv.config.ProjectListURL != "" {
		projectListURL = pv.config.ProjectListURL
	} else {
		projectListURL = "yaml/projectlist.yaml" // Default to local file
	}

	var content string
	var err error

	// Check if it's a URL or local file
	if strings.HasPrefix(projectListURL, "http://") || strings.HasPrefix(projectListURL, "https://") {
		content, err = pv.fetchContent(projectListURL)
	} else {
		data, fileErr := os.ReadFile(projectListURL)
		if fileErr != nil {
			return nil, fileErr
		}
		content = string(data)
	}

	if err != nil {
		return nil, err
	}

	var projectList ProjectListConfig
	if err := yaml.Unmarshal([]byte(content), &projectList); err != nil {
		return nil, fmt.Errorf("failed to parse project list YAML: %v", err)
	}

	var urls []string
	for _, project := range projectList.Projects {
		urls = append(urls, os.ExpandEnv(project.URL))
	}

	return urls, nil
}

// fetchContent fetches content from a URL or local file
func (pv *ProjectValidator) fetchContent(url string) (string, error) {
	// Handle file:// URLs or local paths
	if strings.HasPrefix(url, "file://") || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		filePath := strings.TrimPrefix(url, "file://")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	// Handle HTTP/HTTPS URLs
	resp, err := pv.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// validateProjectStruct validates the project structure
func validateProjectStruct(project Project) []string {
	var errors []string

	// Required fields
	if project.Name == "" {
		errors = append(errors, "name is required")
	}
	if project.Description == "" {
		errors = append(errors, "description is required")
	}

	// Validate maturity log
	if len(project.MaturityLog) == 0 {
		errors = append(errors, "maturity_log is required and cannot be empty")
	} else {
		for i, entry := range project.MaturityLog {
			if entry.Phase == "" {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].phase is required", i))
			}
			if entry.Date.IsZero() {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].date is required", i))
			}
			if entry.Issue == "" {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].issue is required", i))
			}
		}
	}

	// Validate repositories
	if len(project.Repositories) == 0 {
		errors = append(errors, "repositories is required and cannot be empty")
	} else {
		for i, repo := range project.Repositories {
			if !isValidURL(repo) {
				errors = append(errors, fmt.Sprintf("repositories[%d] is not a valid URL: %s", i, repo))
			}
		}
	}

	// Validate URLs
	if project.Website != "" && !isValidURL(project.Website) {
		errors = append(errors, fmt.Sprintf("website is not a valid URL: %s", project.Website))
	}
	if project.Artwork != "" && !isValidURL(project.Artwork) {
		errors = append(errors, fmt.Sprintf("artwork is not a valid URL: %s", project.Artwork))
	}

	// Validate social links
	for platform, url := range project.Social {
		if !isValidURL(url) {
			errors = append(errors, fmt.Sprintf("social.%s is not a valid URL: %s", platform, url))
		}
	}

	// Validate audits
	for i, audit := range project.Audits {
		if audit.Date.IsZero() {
			errors = append(errors, fmt.Sprintf("audits[%d].date is required", i))
		}
		if audit.Type == "" {
			errors = append(errors, fmt.Sprintf("audits[%d].type is required", i))
		}
		if audit.URL == "" {
			errors = append(errors, fmt.Sprintf("audits[%d].url is required", i))
		} else if !isValidURL(audit.URL) {
			errors = append(errors, fmt.Sprintf("audits[%d].url is not a valid URL: %s", i, audit.URL))
		}
	}

	// Validate new fields
	if project.Security != nil {
		if project.Security.Policy != nil && project.Security.Policy.Path == "" {
			errors = append(errors, "security.policy.path is required")
		}
		if project.Security.ThreatModel != nil && project.Security.ThreatModel.Path == "" {
			errors = append(errors, "security.threat_model.path is required")
		}
		if project.Security.Contact != "" {
			if _, err := mail.ParseAddress(project.Security.Contact); err != nil {
				errors = append(errors, fmt.Sprintf("security.contact is not a valid email: %s", project.Security.Contact))
			}
		}
	}

	if project.Governance != nil {
		if project.Governance.Contributing != nil && project.Governance.Contributing.Path == "" {
			errors = append(errors, "governance.contributing.path is required")
		}
		if project.Governance.Codeowners != nil && project.Governance.Codeowners.Path == "" {
			errors = append(errors, "governance.codeowners.path is required")
		}
		if project.Governance.GovernanceDoc != nil && project.Governance.GovernanceDoc.Path == "" {
			errors = append(errors, "governance.governance_doc.path is required")
		}
	}

	if project.Legal != nil {
		if project.Legal.License != nil && project.Legal.License.Path == "" {
			errors = append(errors, "legal.license.path is required")
		}
	}

	if project.Documentation != nil {
		if project.Documentation.Readme != nil && project.Documentation.Readme.Path == "" {
			errors = append(errors, "documentation.readme.path is required")
		}
		if project.Documentation.Support != nil && project.Documentation.Support.Path == "" {
			errors = append(errors, "documentation.support.path is required")
		}
		if project.Documentation.Architecture != nil && project.Documentation.Architecture.Path == "" {
			errors = append(errors, "documentation.architecture.path is required")
		}
		if project.Documentation.API != nil && project.Documentation.API.Path == "" {
			errors = append(errors, "documentation.api.path is required")
		}
	}

	// Validate extensions
	if len(project.Extensions) > 0 {
		extensionErrors := validateExtensions(project)
		errors = append(errors, extensionErrors...)
	}

	return errors
}

// reservedExtensionNames contains names that cannot be used as extension keys
// to prevent conflicts with core project fields
var reservedExtensionNames = map[string]bool{
	"name": true, "description": true, "maturity_log": true,
	"repositories": true, "social": true, "artwork": true,
	"website": true, "mailing_lists": true, "audits": true,
	"schema_version": true, "type": true, "security": true,
	"governance": true, "legal": true, "documentation": true,
	"extensions": true,
}

// validateExtensions validates the extensions section of a project
func validateExtensions(project Project) []string {
	var errors []string

	// Check schema version requirement
	if project.SchemaVersion == "" || !isVersionAtLeast(project.SchemaVersion, SchemaVersionWithExtensions) {
		errors = append(errors, fmt.Sprintf("extensions require schema_version >= %s", SchemaVersionWithExtensions))
		return errors
	}

	for name, ext := range project.Extensions {
		// Validate extension name format (alphanumeric, hyphens, underscores, dots)
		if !isValidExtensionName(name) {
			errors = append(errors, fmt.Sprintf("extensions.%s: invalid name format (use alphanumeric, hyphens, underscores, dots)", name))
		}

		// Check for reserved names
		if reservedExtensionNames[name] {
			errors = append(errors, fmt.Sprintf("extensions.%s: '%s' is a reserved name", name, name))
		}

		// Validate metadata URLs if provided
		if ext.Metadata != nil {
			if ext.Metadata.Homepage != "" && !isValidURL(ext.Metadata.Homepage) {
				errors = append(errors, fmt.Sprintf("extensions.%s.metadata.homepage is not a valid URL", name))
			}
			if ext.Metadata.Repository != "" && !isValidURL(ext.Metadata.Repository) {
				errors = append(errors, fmt.Sprintf("extensions.%s.metadata.repository is not a valid URL", name))
			}
		}
	}

	return errors
}

// isValidExtensionName checks if an extension name follows naming conventions
func isValidExtensionName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

// isVersionAtLeast compares semantic versions (simple comparison)
func isVersionAtLeast(version, minVersion string) bool {
	// Simple version comparison for x.y.z format
	return version >= minVersion
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		return false
	}
	// Check that there's something after the protocol
	if strings.HasPrefix(str, "https://") && len(str) <= 8 {
		return false
	}
	if strings.HasPrefix(str, "http://") && len(str) <= 7 {
		return false
	}
	return true
}

// calculateHash calculates SHA256 hash of content
func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// loadConfig loads configuration from file
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.CacheDir == "" {
		config.CacheDir = ".cache"
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "json"
	}

	return &config, nil
}

// loadCache loads cache from disk
func loadCache(dir string) (*Cache, error) {
	cache := &Cache{
		Entries: make(map[string]CacheEntry),
		dir:     dir,
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	cachePath := filepath.Join(dir, "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return cache, nil // New cache
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &cache.Entries); err != nil {
		return nil, err
	}

	return cache, nil
}

// save saves cache to disk
func (c *Cache) save() error {
	data, err := json.MarshalIndent(c.Entries, "", "  ")
	if err != nil {
		return err
	}

	cachePath := filepath.Join(c.dir, "cache.json")
	return os.WriteFile(cachePath, data, 0644)
}

// GenerateDiff generates a diff report for changed projects
func (pv *ProjectValidator) GenerateDiff(results []ValidationResult) string {
	var diff strings.Builder
	diff.WriteString("Project Validation Report\n")
	diff.WriteString("========================\n\n")

	changedCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Changed || !result.Valid {
			if result.Changed {
				changedCount++
				diff.WriteString(fmt.Sprintf("CHANGED: %s (%s)\n", result.ProjectName, result.URL))
				diff.WriteString(fmt.Sprintf("  Previous Hash: %s\n", result.PreviousHash))
				diff.WriteString(fmt.Sprintf("  Current Hash:  %s\n", result.CurrentHash))
			}
			if !result.Valid {
				errorCount++
				diff.WriteString(fmt.Sprintf("INVALID: %s (%s)\n", result.ProjectName, result.URL))
				for _, err := range result.Errors {
					diff.WriteString(fmt.Sprintf("  - %s\n", err))
				}
			}
			diff.WriteString("\n")
		}
	}

	diff.WriteString(fmt.Sprintf("Summary: %d projects validated, %d changed, %d with errors\n",
		len(results), changedCount, errorCount))

	return diff.String()
}

// NewValidator creates a new validator instance - compatibility alias
func NewValidator(cacheDir string) *ProjectValidator {
	config := &Config{
		ProjectListURL: "yaml/projectlist.yaml",
		CacheDir:       cacheDir,
		OutputFormat:   "text",
	}

	cache, _ := loadCache(cacheDir)

	return &ProjectValidator{
		config: config,
		cache:  cache,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ValidateAll validates all projects from a project list file - compatibility method
func (pv *ProjectValidator) ValidateAll(projectListPath string) ([]ValidationResult, error) {
	if projectListPath != "" {
		pv.config.ProjectListURL = projectListPath
	}
	return pv.ValidateProjects()
}

// FormatResults formats validation results in the specified format
func (pv *ProjectValidator) FormatResults(results []ValidationResult, format string) (string, error) {
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
	case "text":
		return pv.GenerateDiff(results), nil
	default:
		return pv.GenerateDiff(results), nil
	}
}
