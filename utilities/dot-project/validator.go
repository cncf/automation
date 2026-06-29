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
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// githubAdvisoryURLPattern matches GitHub Security Advisory URLs of the form:
// https://github.com/{org}/{repo}/security/advisories/new
var githubAdvisoryURLPattern = regexp.MustCompile(`^https://github\.com/[^/]+/[^/]+/security/advisories/new$`)

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
		client: &http.Client{Timeout: DefaultHTTPTimeout},
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
	for _, projectURL := range projectURLs {
		result, err := pv.validateProject(projectURL)
		if err != nil {
			log.Printf("Error validating project %s: %v", projectURL, err)
			result = ValidationResult{
				URL:         projectURL,
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
	defer func() { _ = resp.Body.Close() }()

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
var SupportedSchemaVersions = []string{"1.0.0"}

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

	// Validate project_lead (optional; each entry must be a valid GitHub handle
	// or a GitHub team reference of the form org/team-name).
	// Accepts a single string (scalar) or a list for projects with multiple leads.
	for i, rawLead := range project.ProjectLeads {
		lead := strings.TrimSpace(rawLead)
		lead = strings.TrimPrefix(lead, "@")
		if lead == "" {
			errors = append(errors, fmt.Sprintf("project_lead[%d] cannot be empty or just '@'", i))
		} else if strings.Contains(lead, "/") {
			parts := strings.Split(lead, "/")
			if len(parts) != 2 {
				errors = append(errors, fmt.Sprintf("project_lead[%d] team format must be org/team-name (got too many segments): %s", i, rawLead))
			} else if parts[0] == "" {
				errors = append(errors, fmt.Sprintf("project_lead[%d] team format requires a non-empty org (expected org/team-name): %s", i, rawLead))
			} else if parts[1] == "" {
				errors = append(errors, fmt.Sprintf("project_lead[%d] team format requires a non-empty team name (expected org/team-name): %s", i, rawLead))
			}
		}
	}

	// Validate slack_channels (optional list of structured channels)
	primarySlackCount := 0
	for i, ch := range project.SlackChannels {
		if ch.Name == "" {
			errors = append(errors, fmt.Sprintf("slack_channels[%d].name is required", i))
		} else if !strings.HasPrefix(ch.Name, "#") {
			errors = append(errors, fmt.Sprintf("slack_channels[%d].name must start with '#', got: %s", i, ch.Name))
		}
		if ch.Link != "" && !isValidURL(ch.Link) {
			errors = append(errors, fmt.Sprintf("slack_channels[%d].link is not a valid URL: %s", i, ch.Link))
		}
		if ch.Primary {
			primarySlackCount++
		}
	}
	if primarySlackCount > 1 {
		errors = append(errors, fmt.Sprintf("at most one slack_channels entry may be marked primary, found %d", primarySlackCount))
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
	for platform, rawURL := range project.Social {
		if !isValidURL(rawURL) {
			errors = append(errors, fmt.Sprintf("social.%s is not a valid URL: %s", platform, rawURL))
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

	// Validate PathRef fields: if a *PathRef is present, its path must not be empty.
	// Top-level PathRef
	errors = append(errors, validatePathRefs([]pathRefCheck{
		{project.Adopters, "adopters"},
	})...)

	// Security section
	if project.Security != nil {
		errors = append(errors, validatePathRefs([]pathRefCheck{
			{project.Security.Policy, "security.policy"},
			{project.Security.ThreatModel, "security.threat_model"},
		})...)

		if project.Security.Contact != nil {
			if project.Security.Contact.Email == "" && project.Security.Contact.AdvisoryURL == "" {
				errors = append(errors, "security.contact must have at least one of email or advisory_url")
			}
			if project.Security.Contact.Email != "" {
				if _, err := mail.ParseAddress(project.Security.Contact.Email); err != nil {
					errors = append(errors, fmt.Sprintf("security.contact.email is not a valid email: %s", project.Security.Contact.Email))
				}
			}
			if project.Security.Contact.AdvisoryURL != "" {
				if !githubAdvisoryURLPattern.MatchString(project.Security.Contact.AdvisoryURL) {
					errors = append(errors, fmt.Sprintf("security.contact.advisory_url must be a valid GitHub Security Advisory URL (https://github.com/{org}/{repo}/security/advisories/new), got: %s", project.Security.Contact.AdvisoryURL))
				}
			}
		}
	}

	// Governance section
	if project.Governance != nil {
		ml := project.Governance.MaintainerLifecycle
		errors = append(errors, validatePathRefs([]pathRefCheck{
			{project.Governance.Contributing, "governance.contributing"},
			{project.Governance.Codeowners, "governance.codeowners"},
			{project.Governance.GovernanceDoc, "governance.governance_doc"},
			{project.Governance.GitVoteConfig, "governance.gitvote_config"},
			{project.Governance.VendorNeutralityStatement, "governance.vendor_neutrality_statement"},
			{project.Governance.DecisionMakingProcess, "governance.decision_making_process"},
			{project.Governance.RolesAndTeams, "governance.roles_and_teams"},
			{project.Governance.CodeOfConduct, "governance.code_of_conduct"},
			{project.Governance.SubProjectList, "governance.sub_project_list"},
			{project.Governance.SubProjectDocs, "governance.sub_project_docs"},
			{project.Governance.ContributorLadder, "governance.contributor_ladder"},
			{project.Governance.ChangeProcess, "governance.change_process"},
			{project.Governance.CommsChannels, "governance.comms_channels"},
			{project.Governance.CommunityCalendar, "governance.community_calendar"},
			{project.Governance.ContributorGuide, "governance.contributor_guide"},
			{ml.OnboardingDoc, "governance.maintainer_lifecycle.onboarding_doc"},
			{ml.ProgressionLadder, "governance.maintainer_lifecycle.progression_ladder"},
			{ml.OffboardingPolicy, "governance.maintainer_lifecycle.offboarding_policy"},
		})...)

		for i, u := range ml.MentoringProgram {
			if !isValidURL(u) {
				errors = append(errors, fmt.Sprintf("governance.maintainer_lifecycle.mentoring_program[%d] is not a valid URL: %s", i, u))
			}
		}
	}

	// Legal section
	if project.Legal != nil {
		errors = append(errors, validatePathRefs([]pathRefCheck{
			{project.Legal.License, "legal.license"},
		})...)

		if project.Legal.IdentityType != nil {
			if project.Legal.IdentityType.HasCLA && !project.Legal.IdentityType.HasDCO && !project.Legal.IdentityType.CLAOnly {
				errors = append(errors, "legal.identity_type: has_cla requires has_dco (CLA cannot be used without DCO; set cla_only: true if this project has an exception)")
			}
			if project.Legal.IdentityType.CLAOnly && !project.Legal.IdentityType.HasCLA {
				errors = append(errors, "legal.identity_type: cla_only requires has_cla to be true")
			}
			errors = append(errors, validatePathRefs([]pathRefCheck{
				{project.Legal.IdentityType.DCOURL, "legal.identity_type.dco_url"},
				{project.Legal.IdentityType.CLAURL, "legal.identity_type.cla_url"},
			})...)
		}
	}

	// Validate package_managers: each key must have at least one non-empty value.
	for key, vals := range project.PackageManagers {
		if len(vals) == 0 {
			errors = append(errors, fmt.Sprintf("package_managers.%s must have at least one value", key))
		}
		for j, v := range vals {
			if strings.TrimSpace(v) == "" {
				errors = append(errors, fmt.Sprintf("package_managers.%s[%d] value must not be empty", key, j))
			}
		}
	}

	// Landscape section
	if project.Landscape != nil {
		if project.Landscape.Category == "" {
			errors = append(errors, "landscape.category is required when landscape section is present")
		}
		if project.Landscape.Subcategory == "" {
			errors = append(errors, "landscape.subcategory is required when landscape section is present")
		}
	}

	// Documentation section
	if project.Documentation != nil {
		errors = append(errors, validatePathRefs([]pathRefCheck{
			{project.Documentation.Readme, "documentation.readme"},
			{project.Documentation.Support, "documentation.support"},
			{project.Documentation.Architecture, "documentation.architecture"},
			{project.Documentation.API, "documentation.api"},
		})...)
	}

	return errors
}

// pathRefCheck pairs a *PathRef with its label for table-driven validation.
type pathRefCheck struct {
	ref   *PathRef
	label string
}

// validatePathRefs checks that each non-nil PathRef has a non-empty path.
func validatePathRefs(checks []pathRefCheck) []string {
	var errs []string
	for _, c := range checks {
		if c.ref != nil && c.ref.Path == "" {
			errs = append(errs, fmt.Sprintf("%s.path is required", c.label))
		}
	}
	return errs
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
		return nil, fmt.Errorf("failed to create cache directory %q: %w", dir, err)
	}

	cachePath := filepath.Join(dir, "cache.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return cache, nil // New cache
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file %q: %w", cachePath, err)
	}

	if err := json.Unmarshal(data, &cache.Entries); err != nil {
		log.Printf("Warning: cache file %q is corrupted, removing and starting fresh: %v", cachePath, err)
		if removeErr := os.Remove(cachePath); removeErr != nil {
			log.Printf("Warning: failed to remove corrupted cache file %q: %v", cachePath, removeErr)
		}
		cache.Entries = make(map[string]CacheEntry)
		return cache, nil
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

	cache, err := loadCache(cacheDir)
	if err != nil {
		log.Printf("Warning: failed to load cache from %s, starting with empty cache: %v", cacheDir, err)
		cache = &Cache{
			Entries: make(map[string]CacheEntry),
			dir:     cacheDir,
		}
	}

	return &ProjectValidator{
		config: config,
		cache:  cache,
		client: &http.Client{Timeout: DefaultHTTPTimeout},
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
