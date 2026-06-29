package projects

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// StringOrSlice — YAML round-trip tests
// ---------------------------------------------------------------------------

func TestStringOrSlice_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    StringOrSlice
		wantErr bool
	}{
		{
			name:  "scalar string",
			input: `lead: "jdoe"`,
			want:  StringOrSlice{"jdoe"},
		},
		{
			name:  "single-item list",
			input: "lead:\n  - jdoe",
			want:  StringOrSlice{"jdoe"},
		},
		{
			name:  "multi-item list",
			input: "lead:\n  - jdoe\n  - jsmith",
			want:  StringOrSlice{"jdoe", "jsmith"},
		},
		{
			name:    "mapping node is rejected",
			input:   "lead:\n  key: value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				Lead StringOrSlice `yaml:"lead"`
			}
			err := yaml.Unmarshal([]byte(tt.input), &out)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected unmarshal error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if len(out.Lead) != len(tt.want) {
				t.Fatalf("length mismatch: got %v, want %v", out.Lead, tt.want)
			}
			for i := range tt.want {
				if out.Lead[i] != tt.want[i] {
					t.Errorf("[%d] got %q, want %q", i, out.Lead[i], tt.want[i])
				}
			}
		})
	}
}

func TestStringOrSlice_MarshalYAML(t *testing.T) {
	tests := []struct {
		name         string
		input        StringOrSlice
		wantScalar   bool   // true → expect a plain string, not a YAML list
		wantContains string // substring that must appear in the marshalled output
		wantOmitted  bool   // true → expect omitempty to drop the field entirely
	}{
		{
			name:         "single element marshals as scalar",
			input:        StringOrSlice{"jdoe"},
			wantScalar:   true,
			wantContains: "jdoe",
		},
		{
			name:         "multiple elements marshal as list",
			input:        StringOrSlice{"jdoe", "jsmith"},
			wantScalar:   false,
			wantContains: "jdoe",
		},
		{
			name:        "empty slice is omitted by omitempty",
			input:       StringOrSlice{},
			wantOmitted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := struct {
				Lead StringOrSlice `yaml:"lead,omitempty"`
			}{Lead: tt.input}

			data, err := yaml.Marshal(wrapper)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}
			out := string(data)

			if tt.wantOmitted {
				if strings.Contains(out, "lead:") {
					t.Errorf("expected field to be omitted, got: %s", out)
				}
				return
			}

			if !strings.Contains(out, tt.wantContains) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantContains, out)
			}

			if tt.wantScalar {
				// A scalar marshals as `lead: jdoe\n` — no list dashes
				if strings.Contains(out, "- ") {
					t.Errorf("expected scalar output, got list: %s", out)
				}
			} else {
				// A sequence marshals with `- ` entries
				if !strings.Contains(out, "- ") {
					t.Errorf("expected list output, got: %s", out)
				}
			}
		})
	}
}

// TestStringOrSlice_RoundTrip marshals and then unmarshals, asserting identity.
func TestStringOrSlice_RoundTrip(t *testing.T) {
	cases := []StringOrSlice{
		{"jdoe"},
		{"jdoe", "jsmith"},
		{"kubernetes/sig-leads", "thockin"},
	}

	for _, original := range cases {
		wrapper := struct {
			Lead StringOrSlice `yaml:"lead"`
		}{Lead: original}

		data, err := yaml.Marshal(wrapper)
		if err != nil {
			t.Fatalf("marshal error for %v: %v", original, err)
		}

		var result struct {
			Lead StringOrSlice `yaml:"lead"`
		}
		if err := yaml.Unmarshal(data, &result); err != nil {
			t.Fatalf("unmarshal error for %v: %v", original, err)
		}

		if len(result.Lead) != len(original) {
			t.Errorf("round-trip length mismatch for %v: got %v", original, result.Lead)
			continue
		}
		for i := range original {
			if result.Lead[i] != original[i] {
				t.Errorf("round-trip mismatch at [%d] for %v: got %q", i, original, result.Lead[i])
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Multiple project_lead validation
// ---------------------------------------------------------------------------

func TestProjectLeadMultipleValidation(t *testing.T) {
	t.Run("two valid handles", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"thockin", "jdoe"}
		errs := validateProjectStruct(project)
		for _, e := range errs {
			if strings.Contains(e, "project_lead") {
				t.Errorf("unexpected project_lead error: %s", e)
			}
		}
	})

	t.Run("valid handle and valid team", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"thockin", "kubernetes/sig-leads"}
		errs := validateProjectStruct(project)
		for _, e := range errs {
			if strings.Contains(e, "project_lead") {
				t.Errorf("unexpected project_lead error: %s", e)
			}
		}
	})

	t.Run("one valid and one invalid — only the invalid index errors", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"thockin", "@"} // index 1 is invalid
		errs := validateProjectStruct(project)

		sawIdx1 := false
		for _, e := range errs {
			if strings.Contains(e, "project_lead[0]") {
				t.Errorf("expected project_lead[0] to be valid, got: %s", e)
			}
			if strings.Contains(e, "project_lead[1]") {
				sawIdx1 = true
			}
		}
		if !sawIdx1 {
			t.Errorf("expected project_lead[1] error for bare '@', got: %v", errs)
		}
	})

	t.Run("all leads invalid produces one error per bad entry", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"@", "org/team/extra"}
		errs := validateProjectStruct(project)

		sawIdx0, sawIdx1 := false, false
		for _, e := range errs {
			if strings.Contains(e, "project_lead[0]") {
				sawIdx0 = true
			}
			if strings.Contains(e, "project_lead[1]") {
				sawIdx1 = true
			}
		}
		if !sawIdx0 {
			t.Error("expected error for project_lead[0]")
		}
		if !sawIdx1 {
			t.Error("expected error for project_lead[1]")
		}
	})
}

// ---------------------------------------------------------------------------
// package_managers multi-value and empty-slice validation
// ---------------------------------------------------------------------------

func TestPackageManagersMultiValue(t *testing.T) {
	t.Run("multiple images for same registry", func(t *testing.T) {
		project := validBaseProject()
		project.PackageManagers = map[string]StringOrSlice{
			"docker": {
				"registry.k8s.io/kube-apiserver",
				"registry.k8s.io/kube-controller-manager",
			},
		}
		errs := validateProjectStruct(project)
		for _, e := range errs {
			if strings.Contains(e, "package_managers") {
				t.Errorf("unexpected package_managers error: %s", e)
			}
		}
	})

	t.Run("empty slice for a key is rejected", func(t *testing.T) {
		project := validBaseProject()
		project.PackageManagers = map[string]StringOrSlice{
			"docker": {},
		}
		errs := validateProjectStruct(project)
		found := false
		for _, e := range errs {
			if strings.Contains(e, "package_managers.docker") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected package_managers.docker error for empty slice, got: %v", errs)
		}
	})

	t.Run("blank string value inside slice is rejected", func(t *testing.T) {
		project := validBaseProject()
		project.PackageManagers = map[string]StringOrSlice{
			"docker": {"valid-image", "   "},
		}
		errs := validateProjectStruct(project)
		found := false
		for _, e := range errs {
			if strings.Contains(e, "package_managers.docker[1]") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected blank-value error for package_managers.docker[1], got: %v", errs)
		}
	})

	t.Run("multiple registries — mixed single and multi", func(t *testing.T) {
		project := validBaseProject()
		project.PackageManagers = map[string]StringOrSlice{
			"docker":   {"img1", "img2"},
			"homebrew": {"my-tool"},
			"fedora":   {"my-project"},
		}
		errs := validateProjectStruct(project)
		for _, e := range errs {
			if strings.Contains(e, "package_managers") {
				t.Errorf("unexpected package_managers error: %s", e)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Staleness — multiple leads and @ stripping
// ---------------------------------------------------------------------------

func TestCheckStalenessMultipleLeads(t *testing.T) {
	staleDate := time.Now().Add(-200 * 24 * time.Hour)

	t.Run("single lead with @ prefix is stripped in output", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"@jdoe"}
		result := CheckStaleness(project, staleDate, 180)
		if result.ProjectLead != "jdoe" {
			t.Errorf("expected '@' stripped to 'jdoe', got %q", result.ProjectLead)
		}
	})

	t.Run("multiple leads are joined with comma", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"thockin", "jsmith"}
		result := CheckStaleness(project, staleDate, 180)
		if result.ProjectLead != "thockin, jsmith" {
			t.Errorf("expected 'thockin, jsmith', got %q", result.ProjectLead)
		}
	})

	t.Run("multiple leads with @ prefixes are all stripped", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = StringOrSlice{"@thockin", "@jsmith"}
		result := CheckStaleness(project, staleDate, 180)
		if result.ProjectLead != "thockin, jsmith" {
			t.Errorf("expected 'thockin, jsmith', got %q", result.ProjectLead)
		}
	})

	t.Run("no leads produces empty string", func(t *testing.T) {
		project := validBaseProject()
		project.ProjectLeads = nil
		result := CheckStaleness(project, staleDate, 180)
		if result.ProjectLead != "" {
			t.Errorf("expected empty project_lead, got %q", result.ProjectLead)
		}
	})
}
