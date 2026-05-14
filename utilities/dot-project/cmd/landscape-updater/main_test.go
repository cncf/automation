package main

import (
	"strings"
	"testing"

	"projects"

	"gopkg.in/yaml.v3"
)

func setupTest(landscapeYAML string) (*yaml.Node, []string) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(landscapeYAML), &root); err != nil {
		panic("Failed to parse test landscape YAML: " + err.Error())
	}
	lines := strings.Split(landscapeYAML, "\n")
	return &root, lines
}

func TestUpdateLandscape(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Orchestration & Management
    subcategories:
      - subcategory:
        name: Scheduling
        items:
          - item:
            name: Kubernetes
            repo_url: https://github.com/kubernetes/kubernetes
            homepage_url: https://old.kubernetes.io
            description: Old description`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:        "Kubernetes",
		Description: "New description",
		Website:     "https://kubernetes.io",
		Repositories: []string{
			"https://github.com/kubernetes/kubernetes",
		},
		Social: map[string]string{
			"twitter":  "https://twitter.com/kubernetesio",
			"slack":    "https://kubernetes.slack.com",
			"linkedin": "https://linkedin.com/company/kubernetes",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update to return true")
	}

	outStr := strings.Join(newLines, "\n")

	if !strings.Contains(outStr, "homepage_url: https://kubernetes.io") {
		t.Errorf("Homepage URL not updated. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "description: New description") {
		t.Errorf("Description not updated. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "twitter: https://twitter.com/kubernetesio") {
		t.Errorf("Twitter not added. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "slack_url: https://kubernetes.slack.com") {
		t.Errorf("Slack URL not added. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "linkedin_url: https://linkedin.com/company/kubernetes") {
		t.Errorf("LinkedIn URL not added. Got:\n%s", outStr)
	}
}

func TestNoChanges(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Orchestration
    subcategories:
      - subcategory:
        name: Scheduling
        items:
          - item:
            name: MyProject
            repo_url: https://github.com/org/myproject
            homepage_url: https://myproject.io
            description: My project description
            twitter: https://twitter.com/myproject`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:        "MyProject",
		Description: "My project description",
		Website:     "https://myproject.io",
		Repositories: []string{
			"https://github.com/org/myproject",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/myproject",
		},
	}

	_, updated := updateLandscape(root, project, lines)
	if updated {
		t.Fatal("Expected no update when values are identical")
	}
}

func TestUpdateExistingField(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            homepage_url: https://old.proj.io
            description: Old desc`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:        "Proj",
		Description: "Old desc",
		Website:     "https://new.proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "homepage_url: https://new.proj.io") {
		t.Errorf("Homepage URL not updated. Got:\n%s", outStr)
	}
	// Description should remain unchanged
	if !strings.Contains(outStr, "description: Old desc") {
		t.Errorf("Description should not have changed. Got:\n%s", outStr)
	}
}

func TestInsertNewField(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            homepage_url: https://proj.io`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:        "Proj",
		Description: "A new description",
		Website:     "https://proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update for new description")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: A new description") {
		t.Errorf("Description not inserted. Got:\n%s", outStr)
	}
	// homepage_url should be unchanged
	if !strings.Contains(outStr, "homepage_url: https://proj.io") {
		t.Errorf("Homepage URL should not have changed. Got:\n%s", outStr)
	}
}

func TestUpdateExtraField(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            homepage_url: https://proj.io
            extra:
              slack_url: https://old-slack.com`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:    "Proj",
		Website: "https://proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
		Social: map[string]string{
			"slack": "https://new-slack.com",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update for extra field")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "slack_url: https://new-slack.com") {
		t.Errorf("Slack URL not updated. Got:\n%s", outStr)
	}
}

func TestInsertNewExtraField(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            homepage_url: https://proj.io
            extra:
              slack_url: https://slack.com`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:    "Proj",
		Website: "https://proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
		Social: map[string]string{
			"slack":    "https://slack.com",
			"linkedin": "https://linkedin.com/proj",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update for new extra field")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "linkedin_url: https://linkedin.com/proj") {
		t.Errorf("LinkedIn URL not inserted. Got:\n%s", outStr)
	}
	// Existing slack_url should be unchanged
	if !strings.Contains(outStr, "slack_url: https://slack.com") {
		t.Errorf("Existing slack_url should not have changed. Got:\n%s", outStr)
	}
}

func TestCreateExtraBlock(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            homepage_url: https://proj.io`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:    "Proj",
		Website: "https://proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
		Social: map[string]string{
			"slack": "https://slack.com/proj",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update for new extra block")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "extra:") {
		t.Errorf("extra: block not created. Got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "slack_url: https://slack.com/proj") {
		t.Errorf("Slack URL not inserted under extra. Got:\n%s", outStr)
	}
}

func TestMultiLineDescriptionReplacement(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Proj
            repo_url: https://github.com/org/proj
            description: >-
              This is a long description that
              spans multiple lines
            homepage_url: https://proj.io`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:        "Proj",
		Description: "Short new description",
		Website:     "https://proj.io",
		Repositories: []string{
			"https://github.com/org/proj",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update for multi-line description")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: Short new description") {
		t.Errorf("Description not replaced. Got:\n%s", outStr)
	}
	// The multi-line continuation should be gone
	if strings.Contains(outStr, "spans multiple lines") {
		t.Errorf("Multi-line continuation lines should have been removed. Got:\n%s", outStr)
	}
	// homepage_url should still be there
	if !strings.Contains(outStr, "homepage_url: https://proj.io") {
		t.Errorf("homepage_url should not have been removed. Got:\n%s", outStr)
	}
}

func TestUnrelatedLinesUntouched(t *testing.T) {
	landscapeYAML := `landscape:
  - category:
    name: Cat
    subcategories:
      - subcategory:
        name: Sub
        items:
          - item:
            name: Other
            repo_url: https://github.com/org/other
            homepage_url: https://other.io
            description: Other project
          - item:
            name: Target
            repo_url: https://github.com/org/target
            homepage_url: https://old.target.io
          - item:
            name: Another
            repo_url: https://github.com/org/another
            homepage_url: https://another.io
            description: Another project`

	root, lines := setupTest(landscapeYAML)

	project := &projects.Project{
		Name:    "Target",
		Website: "https://new.target.io",
		Repositories: []string{
			"https://github.com/org/target",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	// Verify target item was updated
	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "homepage_url: https://new.target.io") {
		t.Errorf("Target homepage_url not updated. Got:\n%s", outStr)
	}

	// Verify other items are byte-for-byte identical
	for i := range newLines {
		if i < len(lines) && lines[i] != newLines[i] {
			// Only the target item's homepage_url line should differ
			if !strings.Contains(lines[i], "old.target.io") {
				t.Errorf("Line %d was modified unexpectedly.\nOriginal: %q\nModified: %q", i+1, lines[i], newLines[i])
			}
		}
	}
}

// realisticLandscapeYAML simulates the real cncf/landscape landscape.yml with
// the exact noise patterns that caused issues in PRs #4820-#4829:
// - Trailing whitespace on various lines
// - Multi-line >- descriptions on unrelated items
// - Empty description with trailing space
// - Multiple items across categories
const realisticLandscapeYAML = `landscape:
  - category:
    name: Provisioning
    subcategories:
      - subcategory:
        name: Automation & Configuration
        items:
          - item:
            name: Meshery
            homepage_url: https://meshery.io
            project: sandbox
            repo_url: https://github.com/meshery/meshery
            logo: meshery.svg
            twitter: https://twitter.com/mesheryio
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            allow_duplicate_repo: true
            extra:
              lfx_slug: meshery
              accepted: '2021-06-22'
              slack_url: https://slack.meshery.io
              linkedin_url: https://www.linkedin.com/showcase/meshery
              youtube_url: https://www.youtube.com/@mesheryio
              clomonitor_name: meshery
          - item:
            name: SomeOther
            homepage_url: https://someother.io
            repo_url: https://github.com/org/someother
            logo: someother.svg
            crunchbase: https://www.crunchbase.com/organization/someother
  - category:
    name: Runtime
    subcategories:
      - subcategory:
        name: Cloud Native Storage
        items:
          - item:
            name: OpenEBS
            homepage_url: https://www.openebs.io/
            project: sandbox
            repo_url: https://github.com/openebs/openebs
            logo: openebs.svg
            twitter: https://twitter.com/openebs
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            extra:
              slack_url: https://cloud-native.slack.com/archives/CSL0PJ8GN
              youtube_url: https://www.youtube.com/channel/UC3ywadaAUQ1FI4YsHZ8wa0g
              clomonitor_name: openebs
  - category:
    name: Orchestration & Management
    subcategories:
      - subcategory:
        name: Scheduling & Orchestration
        items:
          - item:
            name: MSAP.ai
            description: >-
              MSAP.ai(Container Orchestration Platform) is a next-generation application operations platform that simplify complexity and support business innovation in
              cloud-native environment built on Kubernetes. It maximizes the value of Kubernetes adoption for enterprises while minimizing operational costs.
            homepage_url: https://www.openmaru.io
            logo: msap.ai.svg
            crunchbase: https://www.crunchbase.com/organization/openmaru
          - item:
            name: Open Cluster Management
            homepage_url: https://open-cluster-management.io/
            project: sandbox
            repo_url: https://github.com/open-cluster-management-io/ocm
            logo: open-cluster-management.svg
            twitter: https://twitter.com/ocm_io
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            extra:
              slack_url: https://kubernetes.slack.com/archives/C01GE7YSUUF
              youtube_url: https://www.youtube.com/channel/UC7xxOh2jBM5Jfwt3fsBzOZw
              clomonitor_name: ocm
          - item:
            name: kcp
            homepage_url: https://kcp.io
            project: sandbox
            repo_url: https://github.com/kcp-dev/kcp
            logo: kcp.svg
            twitter: https://twitter.com/kcp
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            extra:
              slack_url: http://slack.k8s.io/
              clomonitor_name: kcp
          - item:
            name: plusserver Kubernetes Engine (PSKE)
            description: >-
              The plusserver Kubernetes Engine (PSKE) based on Gardener reduces the complexity in managing multi-cloud environments
              and enables companies to orchestrate their containers and cloud-native applications across a variety of platforms such
              as plusserver's pluscloud open or hyperscalers such as AWS, either by mouseclick or via an API.
            homepage_url: https://www.plusserver.com/en/product/managed-kubernetes/
            logo: plusserver.svg
            crunchbase: https://www.crunchbase.com/organization/plusserver
          - item:
            name: k0s
            homepage_url: https://k0sproject.io/
            project: sandbox
            repo_url: https://github.com/k0sproject/k0s
            logo: k0s-logo-2025.svg
            twitter: https://x.com/k0sproject
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            extra:
              slack_url: http://slack.k8s.io/
              clomonitor_name: k0s
          - item:
            name: ORAS
            homepage_url: https://oras.land/
            project: sandbox
            repo_url: https://github.com/oras-project/oras
            logo: oras.svg
            twitter: https://twitter.com/orasproject
            crunchbase: https://www.crunchbase.com/organization/cloud-native-computing-foundation
            extra:
              slack_url: https://cloud-native.slack.com/messages/oras
              clomonitor_name: oras
  - category:
    name: Members
    subcategories:
      - subcategory:
        name: General
        items:
          - item:
            name: F5 (member)
            homepage_url: https://www.f5.com
            logo: f5-member.svg
            twitter: https://twitter.com/F5
            crunchbase: https://www.crunchbase.com/organization/f5-networks
            joined: '2017-11-01'
          - item:
            name: Breqwatr (member)
            homepage_url: https://www.breqwatr.com
            logo: breqwatr.svg
            crunchbase: https://www.crunchbase.com/organization/breqwatr
            joined: '2026-04-01'
          - item:
            name: Volcano-Kthena
            description:
            homepage_url: https://volcano.sh/
            project: incubating
            repo_url: https://github.com/volcano-sh/kthena
          - item:
            name: vLLM
            logo: ./vllm-logo-text-light.svg
            unnamed_organization: true
            homepage_url: https://docs.vllm.ai/en/latest/
         `

// injectTrailingWhitespace adds trailing spaces to specific lines in the
// landscape YAML to simulate the real cncf/landscape file patterns.
// This is done programmatically because editors strip trailing whitespace.
func injectTrailingWhitespace(lines []string) []string {
	result := make([]string, len(lines))
	copy(result, lines)
	for i, line := range result {
		// Simulate trailing whitespace on crunchbase/joined lines (from PRs)
		if strings.Contains(line, "crunchbase: https://www.crunchbase.com/organization/someother") ||
			strings.Contains(line, "joined: '2017-11-01'") ||
			strings.Contains(line, "joined: '2026-04-01'") {
			result[i] = line + "            "
		}
		// Simulate empty description with trailing space (Volcano-Kthena pattern)
		if strings.TrimSpace(line) == "description:" && strings.Contains(lines[i+1], "homepage_url: https://volcano.sh/") {
			result[i] = line + " "
		}
	}
	return result
}

// setupTestWithNoise parses YAML and injects real-world trailing whitespace
// patterns to simulate the upstream cncf/landscape file.
func setupTestWithNoise(landscapeYAML string) (*yaml.Node, []string) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(landscapeYAML), &root); err != nil {
		panic("Failed to parse test landscape YAML: " + err.Error())
	}
	lines := strings.Split(landscapeYAML, "\n")
	lines = injectTrailingWhitespace(lines)
	return &root, lines
}

// findLine returns the 0-indexed line number containing substr, or -1.
func findLine(lines []string, substr string) int {
	for i, l := range lines {
		if strings.Contains(l, substr) {
			return i
		}
	}
	return -1
}

// TestPR4820_Meshery simulates PR #4820: add description to Meshery.
// The landscape contains trailing whitespace and multi-line >- descriptions
// on unrelated items that must NOT be modified.
func TestPR4820_Meshery(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "Meshery",
		Description: "Infrastructure by Design",
		Website:     "https://meshery.io",
		Repositories: []string{
			"https://github.com/meshery/meshery",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/mesheryio",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")

	// The description should be added
	if !strings.Contains(outStr, "description: Infrastructure by Design") {
		t.Errorf("Meshery description not added. Got:\n%s", outStr)
	}

	// Exactly 1 line should be added (the description)
	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got %d lines changed (original=%d, new=%d)",
			len(newLines)-len(lines), len(lines), len(newLines))
	}

	// All noise patterns must be preserved as-is
	assertNoisePreserved(t, newLines)
}

// TestPR4821_OpenEBS simulates PR #4821: add description to OpenEBS.
func TestPR4821_OpenEBS(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "OpenEBS",
		Description: "Open Source Container Attached Storage, built using Cloud Native Architecture, simplifies running Stateful Applications on Kubernetes",
		Website:     "https://www.openebs.io/",
		Repositories: []string{
			"https://github.com/openebs/openebs",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/openebs",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: Open Source Container Attached Storage") {
		t.Errorf("OpenEBS description not added. Got:\n%s", outStr)
	}

	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got delta=%d", len(newLines)-len(lines))
	}

	assertNoisePreserved(t, newLines)
}

// TestPR4822_OCM simulates PR #4822: add description to Open Cluster Management.
func TestPR4822_OCM(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "Open Cluster Management",
		Description: "Make working with many Kubernetes clusters super easy regardless of where they are deployed",
		Website:     "https://open-cluster-management.io/",
		Repositories: []string{
			"https://github.com/open-cluster-management-io/ocm",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/ocm_io",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: Make working with many Kubernetes clusters") {
		t.Errorf("OCM description not added. Got:\n%s", outStr)
	}

	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got delta=%d", len(newLines)-len(lines))
	}

	assertNoisePreserved(t, newLines)
}

// TestPR4823_kcp simulates PR #4823: add description to kcp.
func TestPR4823_kcp(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "kcp",
		Description: "Kubernetes-like control planes for form-factors and use-cases beyond Kubernetes and container workloads.",
		Website:     "https://kcp.io",
		Repositories: []string{
			"https://github.com/kcp-dev/kcp",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/kcp",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: Kubernetes-like control planes") {
		t.Errorf("kcp description not added. Got:\n%s", outStr)
	}

	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got delta=%d", len(newLines)-len(lines))
	}

	assertNoisePreserved(t, newLines)
}

// TestPR4827_ORAS simulates PR #4827: add description to ORAS.
func TestPR4827_ORAS(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "ORAS",
		Description: "ORAS is the tool for working with OCI Artifacts",
		Website:     "https://oras.land/",
		Repositories: []string{
			"https://github.com/oras-project/oras",
		},
		Social: map[string]string{
			"twitter": "https://twitter.com/orasproject",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: ORAS is the tool for working with OCI Artifacts") {
		t.Errorf("ORAS description not added. Got:\n%s", outStr)
	}

	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got delta=%d", len(newLines)-len(lines))
	}

	assertNoisePreserved(t, newLines)
}

// TestPR4829_k0s simulates PR #4829: add description to k0s.
func TestPR4829_k0s(t *testing.T) {
	root, lines := setupTestWithNoise(realisticLandscapeYAML)

	project := &projects.Project{
		Name:        "k0s",
		Description: "k0s is a CNCF-certified lightweight, Kubernetes distribution with zero dependencies and zero opinion.",
		Website:     "https://k0sproject.io/",
		Repositories: []string{
			"https://github.com/k0sproject/k0s",
		},
		Social: map[string]string{
			"twitter": "https://x.com/k0sproject",
		},
	}

	newLines, updated := updateLandscape(root, project, lines)
	if !updated {
		t.Fatal("Expected update")
	}

	outStr := strings.Join(newLines, "\n")
	if !strings.Contains(outStr, "description: k0s is a CNCF-certified lightweight") {
		t.Errorf("k0s description not added. Got:\n%s", outStr)
	}

	if len(newLines) != len(lines)+1 {
		t.Errorf("Expected exactly 1 line added, got delta=%d", len(newLines)-len(lines))
	}

	assertNoisePreserved(t, newLines)
}

// assertNoisePreserved checks that all the known noise-inducing patterns from
// the real landscape.yml are preserved byte-for-byte in the output.
func assertNoisePreserved(t *testing.T, lines []string) {
	t.Helper()
	outStr := strings.Join(lines, "\n")

	// 1. Trailing whitespace on unrelated entries must be preserved
	// (injected by injectTrailingWhitespace to simulate real landscape.yml)
	trailingWS := []string{
		"organization/someother            ",
		"'2017-11-01'            ",
		"'2026-04-01'            ",
	}
	for _, pattern := range trailingWS {
		if !strings.Contains(outStr, pattern) {
			t.Errorf("Trailing whitespace was stripped from unrelated line.\n  Expected to find: %q", pattern)
		}
	}

	// 2. Multi-line >- descriptions must NOT be collapsed
	multiLinePatterns := []string{
		"description: >-",
		"cloud-native environment built on Kubernetes. It maximizes",
		"and enables companies to orchestrate their containers",
		"as plusserver's pluscloud open or hyperscalers such as AWS",
	}
	for _, pattern := range multiLinePatterns {
		if !strings.Contains(outStr, pattern) {
			t.Errorf("Multi-line description was modified.\n  Expected to find: %q", pattern)
		}
	}

	// 3. Empty description with trailing space (Volcano-Kthena) must be preserved
	volcanoDescLine := findLine(lines, "volcano.sh")
	if volcanoDescLine > 0 {
		prevLine := lines[volcanoDescLine-1]
		if !strings.HasSuffix(prevLine, " ") {
			t.Errorf("Volcano-Kthena empty description trailing space was stripped: %q", prevLine)
		}
	}

	// 4. Trailing whitespace at end of file (vLLM entry) must be preserved
	if !strings.Contains(outStr, "homepage_url: https://docs.vllm.ai/en/latest/\n         ") {
		t.Errorf("Trailing content after vLLM entry was modified")
	}
}

// TestAllPRsNoNoiseRegression runs all 6 projects against the same landscape
// and verifies that ONLY the target project lines differ each time.
func TestAllPRsNoNoiseRegression(t *testing.T) {
	prProjects := []struct {
		name string
		proj projects.Project
	}{
		{"Meshery", projects.Project{
			Name: "Meshery", Description: "Infrastructure by Design",
			Website: "https://meshery.io", Repositories: []string{"https://github.com/meshery/meshery"},
			Social: map[string]string{"twitter": "https://twitter.com/mesheryio"},
		}},
		{"OpenEBS", projects.Project{
			Name: "OpenEBS", Description: "Container Attached Storage",
			Website: "https://www.openebs.io/", Repositories: []string{"https://github.com/openebs/openebs"},
			Social: map[string]string{"twitter": "https://twitter.com/openebs"},
		}},
		{"Open Cluster Management", projects.Project{
			Name: "Open Cluster Management", Description: "Multi-cluster management",
			Website: "https://open-cluster-management.io/", Repositories: []string{"https://github.com/open-cluster-management-io/ocm"},
			Social: map[string]string{"twitter": "https://twitter.com/ocm_io"},
		}},
		{"kcp", projects.Project{
			Name: "kcp", Description: "Kubernetes-like control planes",
			Website: "https://kcp.io", Repositories: []string{"https://github.com/kcp-dev/kcp"},
			Social: map[string]string{"twitter": "https://twitter.com/kcp"},
		}},
		{"ORAS", projects.Project{
			Name: "ORAS", Description: "OCI Artifacts tool",
			Website: "https://oras.land/", Repositories: []string{"https://github.com/oras-project/oras"},
			Social: map[string]string{"twitter": "https://twitter.com/orasproject"},
		}},
		{"k0s", projects.Project{
			Name: "k0s", Description: "Lightweight Kubernetes distribution",
			Website: "https://k0sproject.io/", Repositories: []string{"https://github.com/k0sproject/k0s"},
			Social: map[string]string{"twitter": "https://x.com/k0sproject"},
		}},
	}

	for _, tc := range prProjects {
		t.Run(tc.name, func(t *testing.T) {
			root, origLines := setupTestWithNoise(realisticLandscapeYAML)
			proj := tc.proj

			newLines, updated := updateLandscape(root, &proj, origLines)
			if !updated {
				t.Fatalf("Expected update for %s", tc.name)
			}

			// Count changed lines (excluding inserted lines)
			changedCount := 0
			minLen := len(origLines)
			if len(newLines) < minLen {
				minLen = len(newLines)
			}

			insertedLines := len(newLines) - len(origLines)

			// Find where the insertion happened
			insertionPoint := -1
			for i := 0; i < minLen; i++ {
				if origLines[i] != newLines[i] {
					insertionPoint = i
					break
				}
			}

			if insertionPoint >= 0 {
				// Verify lines after the insertion are unchanged
				for i := insertionPoint; i < len(origLines); i++ {
					j := i + insertedLines
					if j < len(newLines) && origLines[i] != newLines[j] {
						changedCount++
						t.Errorf("Line %d changed unexpectedly for %s.\n  Original: %q\n  Modified: %q",
							i+1, tc.name, origLines[i], newLines[j])
					}
				}
			}

			// Should only have inserted lines (description), no modifications to existing lines
			if changedCount > 0 {
				t.Errorf("%s: %d existing lines were modified (expected 0)", tc.name, changedCount)
			}

			// Should have added exactly 1 line (the description)
			if insertedLines != 1 {
				t.Errorf("%s: expected 1 inserted line, got %d", tc.name, insertedLines)
			}
		})
	}
}

func TestYamlQuoteIfNeeded(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"simple text", "simple text"},
		{"has: colon space", `"has: colon space"`},
		{"has #comment", `"has #comment"`},
		{"*star", `"*star"`},
		{"", "''"},
	}

	for _, tc := range tests {
		result := yamlQuoteIfNeeded(tc.input)
		if result != tc.expected {
			t.Errorf("yamlQuoteIfNeeded(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
