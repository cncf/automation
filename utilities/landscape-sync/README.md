# Landscape Sync Tool

This tool compares projects that have passed GitVote in the [cncf/sandbox](https://github.com/cncf/sandbox) repository against the [CNCF Landscape](https://landscape.cncf.io/) to identify missing projects.

## Purpose

When a project passes the GitVote process in the CNCF sandbox, it should eventually be added to the CNCF Landscape YAML. This tool automates the comparison to:

1. Fetch all issues with the `gitvote/passed` label from cncf/sandbox
2. Fetch the current CNCF Landscape YAML
3. Compare project names to identify gaps
4. Generate reports and suggested YAML entries

## Installation

```bash
# Clone the repository
git clone https://github.com/cncf/automation.git
cd automation/utilities/landscape-sync

# Install dependencies
make deps

# Build the tool
make build
```

## Usage

### Basic Report

Generate a markdown report of missing projects:

```bash
make report
# or
./bin/landscape-sync --output=missing-projects.md
```

### Generate YAML Entries

Generate suggested landscape.yml entries:

```bash
make yaml
# or
./bin/landscape-sync --yaml --output=missing-entries.yaml
```

### JSON Output

Export results as JSON for programmatic use:

```bash
make json
# or
./bin/landscape-sync --json --output=missing-projects.json
```

### Verbose Mode

Run with detailed output:

```bash
./bin/landscape-sync --verbose
```

## GitHub Token

For higher rate limits and access to private information, set a GitHub token:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./bin/landscape-sync
```

## Output Formats

### Markdown Report

The default report includes:
- Table of missing projects with issue numbers, state, and URLs
- Notes about project states and naming variations

### YAML Entries

Generated YAML entries include:
- Project name
- Homepage URL (if available)
- Repository URL (if available)
- Placeholder for logo and Crunchbase
- Project status set to "sandbox"

### JSON Output

Full JSON export of missing project data including:
- Issue details (number, title, state, labels)
- Extracted URLs
- Project description
- Suggested landscape entry

## GitHub Action

This tool is designed to run as a GitHub Action for automated monitoring. See `.github/workflows/landscape-sync.yaml` for the workflow configuration.

## Notes

- Projects with OPEN issue state may still be in the onboarding process
- Some projects may have different names in the landscape vs. their sandbox application
- Manual verification is recommended before creating PRs to add entries
- The tool extracts URLs from issue bodies using pattern matching, which may not always be accurate

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

Apache 2.0 - See [LICENSE](../../LICENSE) for details.
