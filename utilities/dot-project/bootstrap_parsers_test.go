package projects

import (
	"testing"
)

func TestParseCodeowners(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "basic handles",
			content: `* @alice @bob
/docs/ @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "deduplicates handles",
			content: `* @alice @bob
/docs/ @alice @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "skips team references",
			content: `* @org/team-name @alice
/docs/ @org/reviewers @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "skips comment lines",
			content: `# This is a comment
# @notahandle
* @alice`,
			expected: []string{"alice"},
		},
		{
			name: "skips empty lines",
			content: `

* @alice

/docs/ @bob
`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "handles with leading @ stripped",
			content:  `* @alice @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "email addresses are skipped",
			content:  `* @alice someone@example.com @bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "mixed team and user refs",
			content: `* @kubernetes/sig-node-reviewers @dims @thockin
/staging/ @kubernetes/sig-api-machinery-leads @liggitt`,
			expected: []string{"dims", "liggitt", "thockin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCodeowners(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseCodeowners() returned %d handles, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, handle := range got {
				if handle != tt.expected[i] {
					t.Errorf("parseCodeowners()[%d] = %q, want %q", i, handle, tt.expected[i])
				}
			}
		})
	}
}

func TestParseOwnersFile(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		expectedApprovers []string
		expectedReviewers []string
	}{
		{
			name: "basic YAML owners file",
			content: `approvers:
  - alice
  - bob
reviewers:
  - carol
  - dave`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: []string{"carol", "dave"},
		},
		{
			name: "approvers only",
			content: `approvers:
  - alice
  - bob`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: nil,
		},
		{
			name: "reviewers only",
			content: `reviewers:
  - carol`,
			expectedApprovers: nil,
			expectedReviewers: []string{"carol"},
		},
		{
			name:              "empty content",
			content:           "",
			expectedApprovers: nil,
			expectedReviewers: nil,
		},
		{
			name: "handles with @ prefix stripped",
			content: `approvers:
  - "@alice"
  - bob
reviewers:
  - "@carol"`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: []string{"carol"},
		},
		{
			name: "extra whitespace in handles",
			content: `approvers:
  -  alice 
  - bob`,
			expectedApprovers: []string{"alice", "bob"},
			expectedReviewers: nil,
		},
		{
			name: "ignores unknown keys",
			content: `approvers:
  - alice
reviewers:
  - bob
emeritus_approvers:
  - carol
labels:
  - sig/node`,
			expectedApprovers: []string{"alice"},
			expectedReviewers: []string{"bob"},
		},
		{
			name: "handles comment lines in YAML",
			content: `# Top-level owners
approvers:
  - alice
  # inactive: bob
  - carol
reviewers:
  - dave`,
			expectedApprovers: []string{"alice", "carol"},
			expectedReviewers: []string{"dave"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approvers, reviewers := parseOwnersFile(tt.content)
			if len(approvers) != len(tt.expectedApprovers) {
				t.Fatalf("approvers: got %d, want %d\ngot:  %v\nwant: %v",
					len(approvers), len(tt.expectedApprovers), approvers, tt.expectedApprovers)
			}
			for i, h := range approvers {
				if h != tt.expectedApprovers[i] {
					t.Errorf("approvers[%d] = %q, want %q", i, h, tt.expectedApprovers[i])
				}
			}
			if len(reviewers) != len(tt.expectedReviewers) {
				t.Fatalf("reviewers: got %d, want %d\ngot:  %v\nwant: %v",
					len(reviewers), len(tt.expectedReviewers), reviewers, tt.expectedReviewers)
			}
			for i, h := range reviewers {
				if h != tt.expectedReviewers[i] {
					t.Errorf("reviewers[%d] = %q, want %q", i, h, tt.expectedReviewers[i])
				}
			}
		})
	}
}

func TestParseMaintainersFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "plain @handle extraction",
			content: `# Maintainers

@alice - Alice Smith
@bob - Bob Jones
@carol - Carol White`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "markdown table format",
			content: `# Maintainers

| Name | GitHub Handle | Affiliation |
|------|--------------|-------------|
| Alice Smith | @alice | ACME Corp |
| Bob Jones | @bob | BigCo |
| Carol White | @carol | StartupInc |`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "plain text name (handle) format",
			content: `Alice Smith (@alice)
Bob Jones (@bob)
Carol White (@carol)`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "github URLs as handles",
			content: `# Maintainers
- Alice Smith (https://github.com/alice)
- Bob Jones (https://github.com/bob)`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "mixed formats",
			content: `# Project Maintainers

@alice - Lead
Bob Jones (@bob) - Core
| Carol White | @carol | SIG Lead |
- https://github.com/dave`,
			expected: []string{"alice", "bob", "carol", "dave"},
		},
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name: "deduplicates handles",
			content: `@alice - Lead
Alice Smith (@alice) - Also lead`,
			expected: []string{"alice"},
		},
		{
			name: "skips team references",
			content: `@alice
@org/some-team
@bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "handles without @ on lines with GitHub profile URLs",
			content: `- alice https://github.com/alice
- bob https://github.com/bob`,
			expected: []string{"alice", "bob"},
		},
		{
			name: "markdown list with handles",
			content: `## Maintainers
- @alice
- @bob
- @carol`,
			expected: []string{"alice", "bob", "carol"},
		},
		{
			name: "email addresses are not treated as handles",
			content: `@alice - alice@example.com
@bob - bob@example.com`,
			expected: []string{"alice", "bob"},
		},
		{
			name:     "handles with trailing punctuation",
			content:  `@alice, @bob, and @carol are maintainers.`,
			expected: []string{"alice", "bob", "carol"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMaintainersFile(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseMaintainersFile() returned %d handles, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, handle := range got {
				if handle != tt.expected[i] {
					t.Errorf("parseMaintainersFile()[%d] = %q, want %q", i, handle, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractSlackChannel(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "messages URL",
			content:  "Join us at https://cloud-native.slack.com/messages/my-project",
			expected: "#my-project",
		},
		{
			name:     "archives URL",
			content:  "Join us at https://cloud-native.slack.com/archives/my-project",
			expected: "#my-project",
		},
		{
			name:     "channels URL",
			content:  "Join us at https://cloud-native.slack.com/channels/my-project",
			expected: "#my-project",
		},
		{
			name:     "channels URL in markdown link",
			content:  "[#my-project](https://cloud-native.slack.com/channels/my-project)",
			expected: "#my-project",
		},
		{
			name:     "channels URL with trailing slash",
			content:  "https://cloud-native.slack.com/channels/my-project/",
			expected: "#my-project",
		},
		{
			name:     "hash channel on same line as Slack keyword",
			content:  "Slack: #my-project",
			expected: "#my-project",
		},
		{
			name:     "hash channel on same line as CNCF keyword",
			content:  "CNCF Slack channel: #my-project",
			expected: "#my-project",
		},
		{
			name: "channel on separate line near Slack mention",
			content: `## Community
Join our Slack workspace
Channel: #my-project`,
			expected: "#my-project",
		},
		{
			name: "channel on line 3 lines after Slack mention",
			content: `Join us on Slack:
Some info
More info
Channel: #my-project`,
			expected: "#my-project",
		},
		{
			name:     "no slack reference at all",
			content:  "This project is great!",
			expected: "",
		},
		{
			name: "channel too far from Slack mention",
			content: `Join us on Slack
line1
line2
line3
line4
Channel: #my-project`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSlackChannel(tt.content)
			if got != tt.expected {
				t.Errorf("extractSlackChannel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractSlackChannels(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty content",
			content:  "",
			expected: nil,
		},
		{
			name:     "single channel from URL",
			content:  "Join us at https://cloud-native.slack.com/messages/my-project",
			expected: []string{"#my-project"},
		},
		{
			name: "multiple channels from URLs",
			content: `Join [#envoy](https://cloud-native.slack.com/channels/envoy) or
[#envoy-dev](https://cloud-native.slack.com/channels/envoy-dev) for development discussions.`,
			expected: []string{"#envoy", "#envoy-dev"},
		},
		{
			name: "deduplicates same channel from URL and context",
			content: `Slack: #my-project
More info at https://cloud-native.slack.com/messages/my-project`,
			expected: []string{"#my-project"},
		},
		{
			name: "mixed URL and keyword sources",
			content: `Join [#envoy](https://cloud-native.slack.com/channels/envoy)
For mobile: Slack channel #envoy-mobile`,
			expected: []string{"#envoy", "#envoy-mobile"},
		},
		{
			name: "multiple keyword-based channels",
			content: `Slack: #project-general
CNCF Slack: #project-dev`,
			expected: []string{"#project-general", "#project-dev"},
		},
		{
			name: "filters out Slack channel IDs from archives URLs",
			content: `Join us on [Slack](https://cloud-native.slack.com/archives/C01N7PP1THC)
or [#opentelemetry](https://cloud-native.slack.com/messages/opentelemetry)
See also https://cloud-native.slack.com/archives/D03FAB6GN0K`,
			expected: []string{"#opentelemetry"},
		},
		{
			name:     "filters out generic #slack keyword",
			content:  "Join our Slack community at #slack for discussions",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSlackChannels(tt.content)
			if len(got) != len(tt.expected) {
				t.Fatalf("extractSlackChannels() returned %d channels, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.expected), got, tt.expected)
			}
			for i, ch := range got {
				if ch != tt.expected[i] {
					t.Errorf("extractSlackChannels()[%d] = %q, want %q", i, ch, tt.expected[i])
				}
			}
		})
	}
}
