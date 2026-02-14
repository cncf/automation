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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&project); err != nil {
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

// loadProjectList loads the list of project URLs
func (pv *ProjectValidator) loadProjectList() ([]string, error) {
	// For compatibility, check if projectListURL is set, otherwise use a default projectlist.yaml
	var projectListURL string
	if pv.config.ProjectListURL != "" {
		projectListURL = pv.config.ProjectListURL
	} else {
		projectListURL = "testdata/projectlist.yaml" // Default to local file
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

// ValidMaturityPhases lists the allowed maturity phase values
var ValidMaturityPhases = map[string]bool{
	"sandbox":    true,
	"incubating": true,
	"graduated":  true,
	"archived":   true,
}

// SupportedSchemaVersions lists the schema versions this validator supports
var SupportedSchemaVersions = []string{"1.0.0", "1.1.0"}

// ValidateProjectStruct is the exported wrapper for project structure validation
func ValidateProjectStruct(project Project) []string {
	return validateProjectStruct(project)
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

	// Validate slug
	if project.Slug == "" {
		errors = append(errors, "slug is required")
	} else if !isValidSlug(project.Slug) {
		errors = append(errors, fmt.Sprintf("slug must be lowercase alphanumeric with hyphens, got: %s", project.Slug))
	}

	// Validate project_lead (optional but must be non-empty if present)
	if project.ProjectLead != "" {
		lead := strings.TrimSpace(project.ProjectLead)
		lead = strings.TrimPrefix(lead, "@")
		if lead == "" {
			errors = append(errors, "project_lead cannot be empty or just '@'")
		}
	}

	// Validate cncf_slack_channel (optional but must start with # if present)
	if project.CNCFSlackChannel != "" {
		if !strings.HasPrefix(project.CNCFSlackChannel, "#") {
			errors = append(errors, fmt.Sprintf("cncf_slack_channel must start with '#', got: %s", project.CNCFSlackChannel))
		}
	}

	// Validate schema version
	if project.SchemaVersion == "" {
		errors = append(errors, "schema_version is required")
	} else {
		supported := false
		for _, v := range SupportedSchemaVersions {
			if project.SchemaVersion == v {
				supported = true
				break
			}
		}
		if !supported {
			errors = append(errors, fmt.Sprintf("unsupported schema_version: %s (supported: %v)", project.SchemaVersion, SupportedSchemaVersions))
		}
	}

	// Validate maturity log
	if len(project.MaturityLog) == 0 {
		errors = append(errors, "maturity_log is required and cannot be empty")
	} else {
		for i, entry := range project.MaturityLog {
			if entry.Phase == "" {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].phase is required", i))
			}
			if entry.Phase != "" && !ValidMaturityPhases[entry.Phase] {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].phase has invalid value %q (allowed: sandbox, incubating, graduated, archived)", i, entry.Phase))
			}
			if entry.Date.IsZero() {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].date is required", i))
			}
			if entry.Issue == "" {
				errors = append(errors, fmt.Sprintf("maturity_log[%d].issue is required", i))
			}
		}

		// Check chronological ordering
		for i := 1; i < len(project.MaturityLog); i++ {
			if !project.MaturityLog[i-1].Date.IsZero() && !project.MaturityLog[i].Date.IsZero() {
				if project.MaturityLog[i].Date.Before(project.MaturityLog[i-1].Date) {
					errors = append(errors, fmt.Sprintf("maturity_log[%d].date (%s) is before maturity_log[%d].date (%s); entries must be in chronological order",
						i, project.MaturityLog[i].Date.Format("2006-01-02"),
						i-1, project.MaturityLog[i-1].Date.Format("2006-01-02")))
				}
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

	// Validate adopters
	if project.Adopters != nil && project.Adopters.Path == "" {
		errors = append(errors, "adopters.path is required")
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

		// Validate governance DD PathRef fields
		if project.Governance.VendorNeutralityStatement != nil && project.Governance.VendorNeutralityStatement.Path == "" {
			errors = append(errors, "governance.vendor_neutrality_statement.path is required")
		}
		if project.Governance.DecisionMakingProcess != nil && project.Governance.DecisionMakingProcess.Path == "" {
			errors = append(errors, "governance.decision_making_process.path is required")
		}
		if project.Governance.RolesAndTeams != nil && project.Governance.RolesAndTeams.Path == "" {
			errors = append(errors, "governance.roles_and_teams.path is required")
		}
		if project.Governance.CodeOfConduct != nil && project.Governance.CodeOfConduct.Path == "" {
			errors = append(errors, "governance.code_of_conduct.path is required")
		}
		if project.Governance.SubProjectList != nil && project.Governance.SubProjectList.Path == "" {
			errors = append(errors, "governance.sub_project_list.path is required")
		}
		if project.Governance.SubProjectDocs != nil && project.Governance.SubProjectDocs.Path == "" {
			errors = append(errors, "governance.sub_project_docs.path is required")
		}
		if project.Governance.ContributorLadder != nil && project.Governance.ContributorLadder.Path == "" {
			errors = append(errors, "governance.contributor_ladder.path is required")
		}
		if project.Governance.ChangeProcess != nil && project.Governance.ChangeProcess.Path == "" {
			errors = append(errors, "governance.change_process.path is required")
		}
		if project.Governance.CommsChannels != nil && project.Governance.CommsChannels.Path == "" {
			errors = append(errors, "governance.comms_channels.path is required")
		}
		if project.Governance.CommunityCalendar != nil && project.Governance.CommunityCalendar.Path == "" {
			errors = append(errors, "governance.community_calendar.path is required")
		}
		if project.Governance.ContributorGuide != nil && project.Governance.ContributorGuide.Path == "" {
			errors = append(errors, "governance.contributor_guide.path is required")
		}

		// Validate maintainer_lifecycle
		ml := project.Governance.MaintainerLifecycle
		if ml.OnboardingDoc != nil && ml.OnboardingDoc.Path == "" {
			errors = append(errors, "governance.maintainer_lifecycle.onboarding_doc.path is required")
		}
		if ml.ProgressionLadder != nil && ml.ProgressionLadder.Path == "" {
			errors = append(errors, "governance.maintainer_lifecycle.progression_ladder.path is required")
		}
		if ml.OffboardingPolicy != nil && ml.OffboardingPolicy.Path == "" {
			errors = append(errors, "governance.maintainer_lifecycle.offboarding_policy.path is required")
		}
		for i, u := range ml.MentoringProgram {
			if !isValidURL(u) {
				errors = append(errors, fmt.Sprintf("governance.maintainer_lifecycle.mentoring_program[%d] is not a valid URL: %s", i, u))
			}
		}
	}

	if project.Legal != nil {
		if project.Legal.License != nil && project.Legal.License.Path == "" {
			errors = append(errors, "legal.license.path is required")
		}
		if project.Legal.IdentityType != nil {
			if project.Legal.IdentityType.HasCLA && !project.Legal.IdentityType.HasDCO {
				errors = append(errors, "legal.identity_type: has_cla requires has_dco (CLA cannot be used without DCO)")
			}
			if project.Legal.IdentityType.DCOURL != nil && project.Legal.IdentityType.DCOURL.Path == "" {
				errors = append(errors, "legal.identity_type.dco_url.path is required")
			}
			if project.Legal.IdentityType.CLAURL != nil && project.Legal.IdentityType.CLAURL.Path == "" {
				errors = append(errors, "legal.identity_type.cla_url.path is required")
			}
		}
	}

	// Validate landscape
	if project.Landscape != nil {
		if project.Landscape.Category == "" {
			errors = append(errors, "landscape.category is required when landscape section is present")
		}
		if project.Landscape.Subcategory == "" {
			errors = append(errors, "landscape.subcategory is required when landscape section is present")
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
	"extensions": true, "landscape": true, "slug": true,
	"project_lead": true, "cncf_slack_channel": true,
	"package_managers": true, "adopters": true,
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

// isVersionAtLeast compares semantic versions properly
func isVersionAtLeast(version, minVersion string) bool {
	// Parse version strings into comparable parts
	vParts, err := parseVersion(version)
	if err != nil {
		return false
	}
	
	minParts, err := parseVersion(minVersion)
	if err != nil {
		return false
	}
	
	// Compare major, minor, patch in order
	for i := 0; i < 3; i++ {
		if vParts[i] > minParts[i] {
			return true
		}
		if vParts[i] < minParts[i] {
			return false
		}
	}
	
	// All parts are equal
	return true
}

// parseVersion parses a semantic version string into [major, minor, patch]
func parseVersion(version string) ([3]int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("invalid version format: %s", version)
	}
	
	var result [3]int
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, fmt.Errorf("invalid version component: %s", part)
		}
		result[i] = num
	}
	
	return result, nil
}

// isValidSlug checks if a string is a valid project slug (lowercase alphanumeric + hyphens)
func isValidSlug(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	// Must not start or end with hyphen
	if s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	return true
}

// isValidURL checks if a string is a valid HTTP(S) URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	// Host must contain at least one dot (basic domain validation)
	// and must have non-empty labels (reject hosts like "." or ".com")
	if !strings.Contains(u.Host, ".") {
		return false
	}
	parts := strings.SplitN(u.Host, ".", 2)
	if parts[0] == "" || parts[1] == "" {
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
		ProjectListURL: "testdata/projectlist.yaml",
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
