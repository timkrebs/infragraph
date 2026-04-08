# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- TLS support via Vault-style `listener "tcp"` HCL configuration block
- Structured logging with `slog` for API request/response middleware
- Input validation for node IDs and traversal depth parameters
- `--tls-cert` and `--tls-key` CLI flags for server start command
- Standard flag parsing via `flag.FlagSet` for all CLI commands
- Open-source community files (CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, etc.)
- GitHub Actions CI pipeline

### Fixed
- Ignored errors in `db_migrate` for `NodeCount()` and `EdgeCount()`
- BBolt `ForEach` return values now properly propagated
- BBolt `LoadGraph()` unmarshal errors now include key context
- Edge deletion cascade uses correct substring matching
- `writeJSON` errors now logged instead of silently dropped

### Changed
- Makefile updated with proper build targets, ldflags, and output paths
- Health endpoint no longer logs every request

## [0.1.0] - Initial Release

### Added
- Core graph data model with node and edge storage
- BBolt-backed persistent store
- REST API with CRUD operations for nodes and edges
- BFS-based impact analysis
- Static collector for HCL-defined infrastructure graphs
- CLI with server start/stop, status, and graph query commands
- HCL configuration file support
