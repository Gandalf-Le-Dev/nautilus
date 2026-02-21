# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Makefile with test, lint, coverage, vet, and dev environment targets
- GitHub Actions CI pipeline (test, lint, vet)
- API server `StartAsync` method for startup error propagation
- Token bucket rate limiting on API endpoints
- Generic `GetPluginsByType[T]` function for type-safe plugin retrieval
- Comprehensive test suite for config, API, metrics, plugin, core, and database packages
- CHANGELOG following Keep a Changelog format

### Fixed
- API server startup errors now propagated instead of silently logged
- Plugin termination errors now returned and logged instead of discarded
- Debug config endpoint now sanitizes sensitive fields (TLS paths redacted)

### Deprecated
- `PluginRegistry.GetByType(any)` — use `GetPluginsByType[T](registry)` instead
