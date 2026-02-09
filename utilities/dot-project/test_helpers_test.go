package projects

import "time"

// validBaseProject returns a minimal valid Project for use in tests.
// Tests should modify only the fields they care about.
func validBaseProject() Project {
	return Project{
		SchemaVersion: "1.0.0",
		Slug:          "test-project",
		Name:          "Test Project",
		Description:   "A valid test project",
		MaturityLog: []MaturityEntry{
			{
				Phase: "sandbox",
				Date:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				Issue: "https://github.com/cncf/toc/issues/123",
			},
		},
		Repositories: []string{"https://github.com/test/repo"},
	}
}
