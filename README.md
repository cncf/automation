# CNCF Automation

This repository contains automation tools and scripts used by the Cloud Native Computing Foundation (CNCF) and its projects. These tools help streamline various tasks and workflows, making it easier to manage and maintain CNCF projects.

## Overview

The CNCF Automation repository provides various tools that help automate repetitive tasks, standardize workflows, and improve efficiency across CNCF projects. These tools are designed to be reusable and configurable for different project needs.

## Tools and Components

### Self-Hosted Runners

Tools and scripts for managing self-hosted GitHub Actions runners on CNCF's infrastructure (e.g., Oracle Cloud Infrastructure). These runners allow CNCF projects to execute their CI/CD workflows in a controlled environment.

For more information, see the [CI documentation](./ci/README.md).

### Project Status Audit

Cross-checks CNCF project lifecycle data from **LFX PCC** against Landscape, CLOMonitor, maintainers CSV, DevStats, and Artwork, and optionally adds **[LFX Insights](https://insights.linuxfoundation.org/)** **Insights Health** (tier, including **Archived** when shown on Insights) and **Health Score** (number when applicable) when `lfx_insights_health.yaml` is present (informational only; not used for anomaly detection).

See [utilities/audit_project_lifecycle_across_tools/README.md](./utilities/audit_project_lifecycle_across_tools/README.md) for workflows, local usage, and file layout.

### Project Level Moves

A Copilot agent workflow for changing a CNCF project's maturity level (sandbox,
incubating, graduated, archived) consistently across all the repos that track it:
`cncf-landscape`, `clomonitor`, `cncf-foundation`, and `cncf-artwork`.

It is shipped as a VS Code prompt file plus a companion skill:

- Prompt: [.github/prompts/move-project-level.prompt.md](./.github/prompts/move-project-level.prompt.md)
- Skill: [.github/skills/move-project-level/SKILL.md](./.github/skills/move-project-level/SKILL.md)

**Usage:** open a multi-root workspace containing this repo and the four target
repos above, then in Copilot Chat (agent mode) run `/move-project-level` and fill
in the project, new level, and date. The workflow prepares a `move/<slug>-to-<level>`
branch per repo and applies the correct per-repo edit, stopping at the diff for
review (it does not commit or push).

## Contributing

Contributions to improve these automation tools are welcome! Please see our [contributing guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the [Apache License 2.0](LICENSE).