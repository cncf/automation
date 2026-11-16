# Changelog

All notable changes to the landscape MCP server are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Periodic dataset refresh: the dataset is reloaded every 6 hours by default
  (new `--refresh-interval` flag; `0` disables). Remote fetches use
  conditional GETs with the previous `ETag`/`Last-Modified` validators, and a
  failed or unmodified refresh keeps the previous dataset.
- Remote fetch hardening: 64 MiB response cap, 30-second per-attempt timeout,
  and up to 3 attempts with exponential backoff for transient failures
  (network errors, 5xx responses).
- `ping` support and protocol version negotiation: the server speaks MCP
  protocol versions `2025-06-18`, `2025-03-26`, and `2024-11-05`, echoing the
  client's requested version when supported.
- Tool annotations on every tool: `readOnlyHint`, `idempotentHint`,
  `destructiveHint: false`, `openWorldHint: false`.
- Resource templates `landscape://project/{name}` and
  `landscape://member/{name}` for per-entity resource reads.
- New flags for serving other landscapes: `--member-category-match`
  (default `member`), `--tiers` (default `Gold,Silver`), and
  `--landscape-name` / `--landscape-description` now feed tool descriptions,
  resource names, prompts, and the MCP server name.
- Richer tool surface: `get_project_details` accepts a `repo_url` alternative
  and reports ambiguity with candidate lists; `query_projects` gains
  category/subcategory, incubating/accepted date ranges, `oss`/`end_user`/
  `specification` flags, and `parent_project` filters; `query_projects` and
  `query_members` support `sort_by`/`sort_order`; query results include a
  `total` field.
- Makefile with `build`, `test`, `vet`, `fmt-check`, `lint`, and `ci` targets.
- CI gates the module on `go vet` and `go test -race`.
- Version injection: the Docker build sets `serverVersion` from the `VERSION`
  build arg (via `-ldflags -X main.serverVersion=...`); the publish workflow
  passes the tag name or commit SHA.
- Extensive test coverage for protocol handling, MCP compliance, features,
  and refresh/freshness behavior.

### Changed

- The default sort order of `query_projects` and `query_members` is now by
  name, ascending and case-insensitive; sorting is applied before
  `limit`/`offset`, and items without the sort date always sort last.
- Query tools (`query_projects`, `query_members`), `landscape_summary`, and
  `project_metrics` now exclude archived items. Direct lookups
  (`get_project_details`, `get_member_details`, `search_landscape`,
  `compare_projects`) still include them.
- The module path is now
  `github.com/cncf/automation/utilities/landscape-mcp-server` (previously
  `github.com/cncf/landscape2/go/mcp-server`).
- The monolithic `main.go` was split into focused files: `protocol.go`,
  `tools.go`, `resources.go`, `prompts.go`, `mcp_compliance.go`, `fetch.go`,
  `refresh.go`, `sorting.go`, and `config.go`.
- Docker image hardening: static `CGO_ENABLED=0` build with `-trimpath` and
  stripped symbols, and the container now runs as a non-root user (uid 10001).

### Fixed

- Tool execution failures (invalid arguments, ambiguous or missing lookups,
  dataset unavailable) are now returned as MCP tool results flagged
  `isError: true` instead of JSON-RPC protocol errors, as the spec requires.
- An unknown tool name is reported as JSON-RPC error `-32602`.
- Notifications (requests without an `id`) no longer receive responses.
- `get_project_details` prefers an exact case-insensitive name match and
  returns an error listing candidates when a substring match is ambiguous.

### Removed

- The `notifications/dataset_refreshed` hook for triggering a refresh from
  the client; refresh is now driven by `--refresh-interval`.
- The `notifications/serverReady` notification emitted after dataset load.

## [0.2.0] - Baseline

Baseline version. `0.2.0` is the built-in default of `serverVersion`, reported
in the MCP initialize handshake when a release build does not inject a
version. Everything before the changes listed under Unreleased above.

[Unreleased]: https://github.com/cncf/automation/compare/HEAD
[0.2.0]: https://github.com/cncf/automation/tree/main/utilities/landscape-mcp-server
