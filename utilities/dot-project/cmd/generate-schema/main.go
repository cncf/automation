package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type JSONSchema struct {
	Schema               string                        `json:"$schema,omitempty"`
	ID                   string                        `json:"$id,omitempty"`
	Title                string                        `json:"title,omitempty"`
	Description          string                        `json:"description,omitempty"`
	Type                 string                        `json:"type,omitempty"`
	Required             []string                      `json:"required,omitempty"`
	Properties           map[string]JSONSchemaProperty `json:"properties,omitempty"`
	AdditionalProperties *bool                         `json:"additionalProperties,omitempty"`
	Defs                 map[string]JSONSchema         `json:"$defs,omitempty"`
}

type JSONSchemaProperty struct {
	Type                 string                        `json:"type,omitempty"`
	Description          string                        `json:"description,omitempty"`
	Pattern              string                        `json:"pattern,omitempty"`
	Enum                 []string                      `json:"enum,omitempty"`
	Format               string                        `json:"format,omitempty"`
	Items                *JSONSchemaProperty           `json:"items,omitempty"`
	Properties           map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required             []string                      `json:"required,omitempty"`
	Ref                  string                        `json:"$ref,omitempty"`
	AdditionalProperties interface{}                   `json:"additionalProperties,omitempty"`
}

func main() {
	falseVal := false

	schema := JSONSchema{
		Schema:               "https://json-schema.org/draft/2020-12/schema",
		ID:                   "https://github.com/cncf/automation/utilities/dot-project/schema/v1.0.0/project.json",
		Title:                "CNCF Project Metadata",
		Description:          "Schema for CNCF .project repository project.yaml files",
		Type:                 "object",
		AdditionalProperties: &falseVal,
		Required:             []string{"schema_version", "slug", "name", "description", "maturity_log", "repositories"},
		Properties: map[string]JSONSchemaProperty{
			"schema_version":     {Type: "string", Description: "Schema version", Enum: []string{"1.0.0"}},
			"slug":               {Type: "string", Description: "Unique project identifier", Pattern: "^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$"},
			"name":               {Type: "string", Description: "Project display name"},
			"description":        {Type: "string", Description: "One-line project description"},
			"type":               {Type: "string", Description: "Project type (e.g., project, platform, specification)"},
			"project_lead":       {Type: "string", Description: "GitHub handle of primary contact"},
			"cncf_slack_channel": {Type: "string", Description: "CNCF Slack channel", Pattern: "^#"},
			"maturity_log": {
				Type:        "array",
				Description: "Maturity phase transition history (chronological order)",
				Items:       &JSONSchemaProperty{Ref: "#/$defs/MaturityEntry"},
			},
			"repositories": {
				Type:        "array",
				Description: "Repository URLs",
				Items:       &JSONSchemaProperty{Type: "string", Format: "uri"},
			},
			"website":          {Type: "string", Description: "Project website URL", Format: "uri"},
			"artwork":          {Type: "string", Description: "Artwork/logo URL", Format: "uri"},
			"social":           {Type: "object", Description: "Social platform URLs", AdditionalProperties: JSONSchemaProperty{Type: "string", Format: "uri"}},
			"mailing_lists":    {Type: "array", Description: "Mailing list addresses", Items: &JSONSchemaProperty{Type: "string"}},
			"audits":           {Type: "array", Description: "Security/performance audits", Items: &JSONSchemaProperty{Ref: "#/$defs/Audit"}},
			"adopters":         {Ref: "#/$defs/PathRef", Description: "Link to ADOPTERS.md or adopters list"},
			"package_managers": {Type: "object", Description: "Registry identifiers (e.g., npm package name, Docker Hub image)", AdditionalProperties: JSONSchemaProperty{Type: "string"}},
			"security":         {Ref: "#/$defs/SecurityConfig", Description: "Security configuration"},
			"governance":       {Ref: "#/$defs/GovernanceConfig", Description: "Governance configuration"},
			"legal":            {Ref: "#/$defs/LegalConfig", Description: "Legal configuration"},
			"documentation":    {Ref: "#/$defs/DocumentationConfig", Description: "Documentation configuration"},
			"landscape":        {Ref: "#/$defs/LandscapeConfig", Description: "CNCF Landscape location"},
		},
		Defs: map[string]JSONSchema{
			"MaturityEntry": {
				Type:     "object",
				Required: []string{"phase", "date", "issue"},
				Properties: map[string]JSONSchemaProperty{
					"phase": {Type: "string", Enum: []string{"sandbox", "incubating", "graduated", "archived"}},
					"date":  {Type: "string", Format: "date-time"},
					"issue": {Type: "string", Description: "TOC issue URL"},
				},
			},
			"Audit": {
				Type:     "object",
				Required: []string{"date", "type", "url"},
				Properties: map[string]JSONSchemaProperty{
					"date": {Type: "string", Format: "date-time"},
					"type": {Type: "string", Description: "Audit type (e.g., security, performance)"},
					"url":  {Type: "string", Format: "uri"},
				},
			},
			"PathRef": {
				Type:     "object",
				Required: []string{"path"},
				Properties: map[string]JSONSchemaProperty{
					"path": {Type: "string", Description: "File path or URL"},
				},
			},
			"SecurityConfig": {
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"policy":       {Ref: "#/$defs/PathRef"},
					"threat_model": {Ref: "#/$defs/PathRef"},
					"contact":      {Type: "string", Format: "email", Description: "Security contact email"},
				},
			},
			"GovernanceConfig": {
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"contributing":                {Ref: "#/$defs/PathRef"},
					"codeowners":                  {Ref: "#/$defs/PathRef"},
					"governance_doc":              {Ref: "#/$defs/PathRef"},
					"gitvote_config":              {Ref: "#/$defs/PathRef"},
					"vendor_neutrality_statement": {Ref: "#/$defs/PathRef", Description: "Vendor neutrality statement"},
					"decision_making_process":     {Ref: "#/$defs/PathRef", Description: "Decision-making process documentation"},
					"roles_and_teams":             {Ref: "#/$defs/PathRef", Description: "Roles and teams documentation"},
					"code_of_conduct":             {Ref: "#/$defs/PathRef", Description: "Code of conduct"},
					"sub_project_list":            {Ref: "#/$defs/PathRef", Description: "Subproject listing"},
					"sub_project_docs":            {Ref: "#/$defs/PathRef", Description: "Subproject documentation"},
					"contributor_ladder":          {Ref: "#/$defs/PathRef", Description: "Contributor ladder documentation"},
					"change_process":              {Ref: "#/$defs/PathRef", Description: "Change process documentation"},
					"comms_channels":              {Ref: "#/$defs/PathRef", Description: "Communication channels listing"},
					"community_calendar":          {Ref: "#/$defs/PathRef", Description: "Community calendar"},
					"contributor_guide":           {Ref: "#/$defs/PathRef", Description: "Contributor guide"},
					"maintainer_lifecycle":        {Ref: "#/$defs/MaintainerLifecycle", Description: "Maintainer lifecycle documentation"},
				},
			},
			"LegalConfig": {
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"license":       {Ref: "#/$defs/PathRef"},
					"identity_type": {Ref: "#/$defs/IdentityType", Description: "Contributor identity agreement (DCO, CLA, or none)"},
				},
			},
			"IdentityType": {
				Type:        "object",
				Description: "Contributor identity agreements. DCO can be used alone or with CLA. CLA requires DCO.",
				Properties: map[string]JSONSchemaProperty{
					"has_dco": {Type: "boolean", Description: "Whether the project uses DCO"},
					"has_cla": {Type: "boolean", Description: "Whether the project uses CLA (requires DCO)"},
					"dco_url": {Ref: "#/$defs/PathRef", Description: "Link to DCO document"},
					"cla_url": {Ref: "#/$defs/PathRef", Description: "Link to CLA document"},
				},
			},
			"DocumentationConfig": {
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"readme":       {Ref: "#/$defs/PathRef"},
					"support":      {Ref: "#/$defs/PathRef"},
					"architecture": {Ref: "#/$defs/PathRef"},
					"api":          {Ref: "#/$defs/PathRef"},
				},
			},
			"LandscapeConfig": {
				Type:     "object",
				Required: []string{"category", "subcategory"},
				Properties: map[string]JSONSchemaProperty{
					"category":    {Type: "string", Description: "CNCF Landscape category"},
					"subcategory": {Type: "string", Description: "CNCF Landscape subcategory"},
				},
			},
			"MaintainerLifecycle": {
				Type:        "object",
				Description: "Maintainer lifecycle documentation",
				Properties: map[string]JSONSchemaProperty{
					"onboarding_doc":     {Ref: "#/$defs/PathRef", Description: "URL to maintainer onboarding documentation"},
					"progression_ladder": {Ref: "#/$defs/PathRef", Description: "URL to maintainer advancement path documentation (committer → maintainer → lead)"},
					"mentoring_program":  {Type: "array", Description: "URLs to mentoring and onboarding support program documentation", Items: &JSONSchemaProperty{Type: "string", Format: "uri"}},
					"offboarding_policy": {Ref: "#/$defs/PathRef", Description: "URL to emeritus/offboarding policy documentation"},
				},
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding schema: %v\n", err)
		os.Exit(1)
	}
}
