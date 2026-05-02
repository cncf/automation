# Contributing to cncf-automation

Thank you for your interest in contributing to the CNCF Automation repository! This guide will help you understand how to contribute effectively.

## Getting Started

### Prerequisites
- GitHub account
- Git installed locally
- Familiarity with the repository structure (see [README.md](../README.md))

### Setting Up

1. **Fork the repository**
   ```bash
   git clone https://github.com/YOUR-USERNAME/cncf-automation.git
   cd cncf-automation
   ```

2. **Create a branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow the existing code style
   - Write clear commit messages
   - Test your changes locally

4. **Push and create a Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```

---

## Labeling Issues and PRs

We use automated labeling via slash commands to organize and prioritize work. **All new issues and PRs should be labeled**.

### Quick Reference

Comment on an issue or PR with:

```
/kind bug
/priority high
/area ci
/status in-progress
```

### Full Guide

See our comprehensive **[Label & ChatOps Guide](./README_LABELING.md)** for:
- All available labels and their meanings
- Slash command syntax
- Auto-labeling behavior
- Best practices for labeling

### Important Labels

- **`kind/*`** — Mark the type of work (bug, enhancement, docs, chore, etc.)
- **`priority/*`** — Indicate urgency (critical, high, medium, low)
- **`area/*`** — Specify affected codebase area (ci, utilities, infrastructure, etc.)
- **`status/*`** — Track current state (needs-review, in-progress, blocked)

---

## Commit Guidelines

- Write clear, descriptive commit messages
- Start with a present-tense verb ("Add", "Fix", "Update", not "Added", "Fixed", "Updated")
- Reference related issues: `Fixes #123` or `Related to #456`
- Keep commits atomic and focused on a single change

### Example

```
Fix CI pipeline failure on ARM64 architecture

- Update cloudrunners Dockerfile for multi-arch support
- Add ARM64 tests to GitHub Actions workflow
- Document architecture-specific considerations

Fixes #123
```

---

## Code Review Process

1. **Automated Checks**
   - GitHub Actions workflows run automatically on every PR
   - Syntax checks, linting, and tests are performed
   - All checks must pass before merge

2. **Manual Review**
   - At least one maintainer review is required
   - Reviewers will provide feedback via PR comments
   - Address feedback and update your branch

3. **Merge**
   - Once approved and all checks pass, the PR can be merged
   - Please squash commits if requested by reviewers

---

## Code Style

### Go
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Run `go vet ./...` before submitting

### Python
- Use 4 spaces for indentation
- Follow [PEP 8](https://pep8.org/) style guide
- Use type hints where possible

### YAML
- Use 2 spaces for indentation
- Keep line length under 120 characters
- Use consistent quote style

---

## Testing

Before submitting a PR, ensure:

1. **Unit tests pass**
   ```bash
   go test ./...          # For Go
   python -m pytest       # For Python
   ```

2. **Local testing**
   - Test your changes locally in a test branch
   - Verify all workflows execute correctly

3. **No breaking changes**
   - Document any API or configuration changes
   - Consider backward compatibility

---

## Reporting Issues

When reporting a bug or suggesting a feature:

1. **Check for duplicates** — Search existing issues first
2. **Be specific** — Provide clear, detailed descriptions
3. **Include context** — Share steps to reproduce, error messages, environment info
4. **Use labels** — Tag issues with `/kind`, `/priority`, `/area`

### Issue Template

```
/kind bug
/priority medium
/area ci

## Description
Brief description of the issue

## Steps to Reproduce
1. First step
2. Second step
3. Expected result vs actual result

## Environment
- OS: Linux/macOS/Windows
- Go version: 1.XX
- Python version: 3.XX
```

---

## Documentation

- Update documentation when adding features
- Keep README files accurate and current
- Add comments to complex code sections
- Document configuration options and environment variables

---

## Getting Help

- **Questions**: Open an issue with the `kind/question` label
- **Discussion**: Use issue comments for discussion
- **Chat**: Check existing issues for similar questions

---

## License

By contributing, you agree that your contributions will be licensed under the same license as the repository (see LICENSE file).

---

## Code of Conduct

Please note that this project is governed by the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). By participating, you are expected to uphold this code.

---

## Thank You!

Your contributions help make the CNCF automation infrastructure better for everyone. We appreciate your effort and patience!
