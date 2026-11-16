# Landscape MCP Server

Easily ask questions of the CNCF (Or other!) Landscape. 

## Installation

Add this server to your mcp.json (or equivalent):

```
"cncf-landscape": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "ghcr.io/cncf/landscape-mcp-server:main",
        "--data-url",
        "https://landscape.cncf.io/data/full.json"
      ]
    }
```

## Examples

- How many CNCF projects graduated in 2024?
- When did OpenTelemetry reach incubating?
- What CNCF projects moved levels in 2025?

## Tools

The server advertises the following tools over MCP. Every tool carries the
annotations `readOnlyHint: true`, `destructiveHint: false`,
`idempotentHint: true`, and `openWorldHint: false`.

Query tools (`query_projects`, `query_members`) exclude archived items, as do
`landscape_summary` and `project_metrics`. Direct lookups do not:
`get_project_details`, `get_member_details`, `search_landscape`, and
`compare_projects` can all see archived entries.

For the query tools, `limit` defaults to 100 (values `<= 0` use 100) and is
clamped to a maximum of 500. Results include a `total` field with the number
of matches ignoring `limit`/`offset`. Default sorting is by name, ascending
and case-insensitive; use `sort_by`/`sort_order` to change it. Sorting is
applied before `limit`/`offset`, and items without the sort date always sort
last.

| Tool | Purpose | Arguments |
| --- | --- | --- |
| `query_projects` | Query projects filtered by maturity, dates, name, flags, and parent project | `maturity` (string), `name` (string, substring), `category` (string), `subcategory` (string), `graduated_from` / `graduated_to` / `incubating_from` / `incubating_to` / `accepted_from` / `accepted_to` (string, `YYYY-MM-DD`), `oss` / `end_user` / `specification` (boolean), `parent_project` (string, exact match), `sort_by` (enum: `name`, `accepted_at`, `incubating_at`, `graduated_at`; default `name`), `sort_order` (enum: `asc` default, `desc`), `limit` (integer; default 100, max 500), `offset` (integer; default 0) |
| `query_members` | Query members filtered by tier and join dates | `tier` (string), `category` (string), `joined_from` / `joined_to` (string, `YYYY-MM-DD`), `sort_by` (enum: `name`, `joined_at`; default `name`), `sort_order` (enum: `asc` default, `desc`), `limit` (integer; default 100, max 500), `offset` (integer; default 0) |
| `get_project_details` | Detailed info on one project, by name or by repository/homepage URL | `name` (string; exact case-insensitive match preferred, otherwise unique substring; takes precedence), `repo_url` (string; alternative to `name`, trailing slash and `.git` ignored). One of the two is required. |
| `get_member_details` | Detailed info on one member by name | `name` (string, required; case-insensitive) |
| `list_categories` | List all categories and subcategories with item counts | none |
| `landscape_summary` | High-level statistical overview of the landscape | none |
| `search_landscape` | Full-text search across items by name, description, category, subcategory, or homepage URL | `query` (string, required; case-insensitive substring), `limit` (integer; default 20, max 100) |
| `compare_projects` | Side-by-side comparison of two or more projects | `names` (array of strings, required; minimum 2) |
| `funding_analysis` | Analyze Crunchbase funding data for landscape members | `tier` (string), `year` (integer) |
| `geographic_distribution` | Geographic breakdown of members by Crunchbase location data | `group_by` (enum: `country` default, `region`, `city`) |
| `project_metrics` | Project counts and progress metrics | `metric` (string, required; enum: `incubating_project_count`, `sandbox_projects_joined_this_year`, `projects_graduated_last_year`) |
| `membership_metrics` | Membership activity and funding metrics | `metric` (string, required; enum: `gold_members_joined_this_year`, `silver_members_joined_this_year`, `silver_members_raised_last_month`, `gold_members_raised_this_year`) |

Tool failures (bad arguments, unknown names, ambiguous lookups, dataset
unavailable) are returned as normal MCP tool results flagged with
`isError: true`, not as JSON-RPC protocol errors.

## Resources and prompts

Resources (all `application/json`):

- `landscape://categories` — complete category and subcategory tree with item counts
- `landscape://summary` — high-level landscape statistics and overview

Resource templates:

- `landscape://project/{name}` — detailed information about a single project, looked up by name
- `landscape://member/{name}` — detailed information about a single member, looked up by name

Prompts:

- `analyze_landscape` — analyze the current state of the landscape (no arguments)
- `compare_projects` — compare specific projects (required argument `project_names`, a comma-separated list)
- `landscape_changes_report` — report promotions, new sandbox projects, and archived projects (optional argument `since`, a `YYYY-MM-DD` date; defaults to 12 months ago)

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--data-file` | (empty) | Path to a local landscape full dataset JSON file. Takes precedence over `--data-url`. |
| `--data-url` | `https://landscape.cncf.io/data/full.json` | URL of the landscape full dataset JSON file. |
| `--landscape-name` | `CNCF` | Name of the landscape (e.g. `CNCF`). Used in tool descriptions, resource names, prompts, and the MCP server name. |
| `--landscape-description` | `Cloud Native Computing Foundation` | Description of the landscape. |
| `--member-category-match` | `member` | Case-insensitive substring of an item's category that marks it as a member. |
| `--tiers` | `Gold,Silver` | Comma-separated membership tiers backing the `gold_`/`silver_` metrics, high tier first. |
| `--refresh-interval` | `6h` | How often to reload the dataset from its source. `0` disables periodic refresh. |

### Dataset refresh

The dataset is loaded once at startup; the server answers requests as soon as
the load finishes (requests wait up to 60 seconds for it). When
`--refresh-interval` is positive, the dataset is then reloaded periodically:

- Remote sources are re-fetched with a conditional GET using the previous
  response's `ETag`/`Last-Modified` validators; a `304 Not Modified` leaves
  the current dataset in place.
- On any failure the previous dataset is kept and serving continues.
- Remote bodies are capped at 64 MiB, each attempt has a 30-second timeout,
  and transient failures (network errors, 5xx responses) are retried up to 3
  times with exponential backoff (1s, 2s).
- A successful refresh atomically swaps in the new dataset; in-flight requests
  keep their snapshot.

## Running

### Docker

The published image is `ghcr.io/cncf/landscape-mcp-server`, built and pushed
by the `publish-landscape-mcp-server` GitHub Actions workflow. It is a
multi-platform image (linux/amd64, linux/arm64) that runs as a non-root user.

The workflow tags each push with `type=sha` (a `sha-<short>` tag), the branch
name, and the PR number. Branch tags such as `:main` are moving tags; pin to
a `sha-<short>` tag or a full digest for reproducible deployments:

```
ghcr.io/cncf/landscape-mcp-server:sha-<commit>
ghcr.io/cncf/landscape-mcp-server@sha256:<digest>
```

Images are signed with cosign (keyless, via Sigstore) by the publish workflow.
You can verify a signature with `cosign verify` against the image digest.

### Local build

```sh
make build                      # produces ./landscape2-mcp-server
go build -o landscape2-mcp-server
```

Run the binary directly, or point it at a local dataset file for offline use:

```sh
./landscape2-mcp-server --data-file ./full.json
```

`make ci` runs the same checks as CI: `gofmt` verification, `go vet`, and the
test suite with the race detector.

## Naming

Three names show up around this server and they are intentionally different:

- The binary is `landscape2-mcp-server` (historical name, kept).
- The container image is `ghcr.io/cncf/landscape-mcp-server`.
- The MCP `serverInfo.name` reported in the initialize handshake is
  `<lowercased --landscape-name>-landscape-mcp-server`, e.g.
  `cncf-landscape-mcp-server` with the default configuration.

The `cncf-landscape` key in the installation snippet above is just the
client-side name in your MCP configuration and can be anything you like.

## Using other landscapes

The server is not hardcoded to the CNCF landscape:

- `--data-url` / `--data-file` select any landscape dataset in the
  landscape2 `full.json` format.
- `--landscape-name` / `--landscape-description` relabel tool descriptions,
  resource names, prompts, and the MCP server name.
- `--member-category-match` changes how member items are recognized: an item
  counts as a member when its category contains the given substring
  (case-insensitive). Default `member` matches the CNCF dataset's
  `CNCF Members` category.
- `--tiers` names the two membership tiers behind the tier-specific metrics:
  the first entry backs the `gold_members_*` metrics and the second backs
  `silver_members_*`. Other tiers remain usable as free-form filters in
  `query_members` and `funding_analysis`.

One limitation: the metric names themselves (`gold_members_joined_this_year`,
`silver_members_raised_last_month`, etc.) are CNCF-membership-tier specific.
`--tiers` changes which tiers they measure, but the names stay as they are.

## Protocol notes

- Transport is stdio: newline-delimited JSON-RPC 2.0 on stdin/stdout, with
  logs on stderr. The maximum accepted line length is 16 MiB.
- Supported MCP protocol versions, negotiated during `initialize`:
  `2025-06-18`, `2025-03-26`, `2024-11-05`. The server echoes the client's
  requested version when supported and otherwise answers with its latest
  version.
- `ping` is supported and returns an empty result.
- Declared capabilities: `tools`, `resources`, `prompts`.

## Development

- `make build` — build the binary
- `make test` — run tests with the race detector
- `make vet` — run `go vet`
- `make fmt-check` — verify `gofmt` cleanliness
- `make lint` — run `golangci-lint` if installed
- `make ci` — `fmt-check` + `vet` + `test`

The module is stdlib-only; there are no external dependencies and no `go.sum`.

The version reported in the initialize handshake comes from the
`serverVersion` variable (default `0.2.0`). Release builds override it with:

```sh
go build -ldflags "-X main.serverVersion=<version>"
```

The Docker image does this automatically from the `VERSION` build arg, which
the publish workflow sets to the tag name or commit SHA.

## License

See the [LICENSE](../../LICENSE) file at the repository root.
