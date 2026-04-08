# Contributing to InfraGraph

**First:** if you're unsure or afraid of _anything_, just ask or submit the
issue or pull request anyway. You won't be yelled at for giving it your best
effort. The worst that can happen is that you'll be politely asked to change
something. We appreciate all contributions, and don't want a wall of rules to
get in the way of that.

That said, if you want to ensure that a pull request is likely to be merged,
talk to us! You can find out our thoughts and ensure that your contribution
won't clash with InfraGraph's direction. A great way to do this is via
[GitHub Discussions](https://github.com/timkrebs/infragraph/discussions).

## Issues

### Reporting a Bug

- Make sure you test against the latest released version. It is possible we
  already fixed the bug you're experiencing.
- Provide steps to reproduce the issue, including the expected results and the
  actual results. Please provide text, not screenshots.
- Include the InfraGraph version (`infragraph --version`), Go version, and OS.
- If you experienced a panic, please create a
  [gist](https://gist.github.com) of the entire crash log.

### Suggesting a Feature

- Open a [Feature Request](https://github.com/timkrebs/infragraph/issues/new?template=feature_request.md)
  issue describing the problem, your proposed solution, and alternatives
  you considered.
- For large changes (new collector, architectural change), please open a
  discussion first so we can align on the approach.

### Issue Lifecycle

1. The issue is reported.
2. The issue is verified and categorized via labels (e.g. `bug`, `enhancement`).
3. Unless it is critical, the issue may be left for a period to give outside
   contributors a chance to address it.
4. The issue is addressed in a pull request. The PR description references the
   issue number.
5. The issue is closed when the PR is merged.

## Pull Requests

When submitting a PR you should reference an existing issue. If no issue
already exists, please create one first. This can be skipped for trivial
changes like fixing typos.

### Before You Start

1. Fork the repository and create your branch from `main`.
2. Run `make tidy` to format your code.
3. Run `make test` to ensure all tests pass.
4. Run `make audit` to run the full quality control suite.

### PR Checklist

- [ ] Code compiles without errors (`go build ./...`).
- [ ] All existing tests pass (`make test`).
- [ ] New code has appropriate test coverage.
- [ ] Code passes `go vet` and `golangci-lint`.
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).
- [ ] PR description explains _what_ changed and _why_.

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `perf`.

Examples:
- `feat(collector): add Kubernetes pod watcher`
- `fix(store): handle corrupt bbolt entries on load`
- `docs: add troubleshooting section to README`

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions focused and small.
- Exported symbols must have doc comments.
- Error messages should be lowercase and not end with punctuation.
- Wrap errors with `fmt.Errorf("context: %w", err)`.

### New Collectors

If you're adding a new collector:

1. Implement the `collector.Collector` interface.
2. Add integration tests with a mock or containerized target.
3. Add a config block example in `example/infragraph.hcl`.
4. Document the collector in the README plugin table.

## Development Setup

```bash
# Clone
git clone https://github.com/timkrebs/infragraph.git
cd infragraph

# Build
make build

# Test
make test

# Full quality suite
make audit

# Format & tidy
make tidy
```

## Getting Help

- **GitHub Discussions** — For questions and ideas
- **Issues** — For bug reports and feature requests

Thank you for contributing to InfraGraph!
