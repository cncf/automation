# GitHub Labeler

A Go program that automatically labels GitHub issues and pull requests based on configurable rules defined in a `labels.yaml` file.

## Rule Types Supported

### 1. Match Rules (`kind: match`)
Process slash commands in comments:
```yaml
- name: apply-triage
  kind: match
  spec:
    command: "/triage"
    matchList: ["valid", "duplicate", "needs-information", "not-planned"]
  actions:
  - kind: remove-label
    spec:
      match: needs-triage
  - kind: apply-label
    spec:
      label: "triage/{{ argv.0 }}"
```

### 2. Label Rules (`kind: label`)
Apply labels based on existing label presence:
```yaml
- name: needs-triage
  kind: label
  spec:
    match: "triage/*"
    matchCondition: NOT
  actions:
  - kind: apply-label
    spec:
      label: "needs-triage"
```

### 3. File Path Rules (`kind: filePath`)
Apply labels based on changed file paths:
```yaml
- name: charter
  kind: filePath
  spec:
    matchPath: "tags/*/charter.md"
  actions:
  - kind: apply-label
    spec:
      label: toc
```

## Action Types

### Apply Label
```yaml
- kind: apply-label
  spec:
    label: "label-name"
```

### Remove Label
```yaml
- kind: remove-label
  spec:
    match: "label-pattern"  # Supports wildcards like "triage/*"
```

## Testing

Run tests:
```bash
go test -v
```

## Usage

### CLI
```bash
./labeler <labels_url> <owner> <repo> <issue_number> <comment_body> <changed_files>
```

### GitHub Actions Workflow
The included workflow automatically runs the labeler on issue comments.

## Configuration

The labeler reads configuration from a `labels.yaml` file that defines:

- **Label definitions** with colors and descriptions
- **Rule sets** for automated labeling
- **Global settings** for auto-creation/deletion

## Development

### Adding New Rule Types

1. Add new rule processing function in `labeler.go`
2. Update `processRule()` to handle the new rule type
3. Add corresponding tests in test files

### Mock Testing

The `MockGitHubClient` provides comprehensive mocking for testing complex scenarios without hitting the GitHub API.
