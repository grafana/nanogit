# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1](https://github.com/grafana/nanogit/compare/v0.1.0...v0.1.1) (2025-11-11)


### Bug Fixes

* enable CI checks on CHANGELOG PRs ([#113](https://github.com/grafana/nanogit/issues/113)) ([41a1797](https://github.com/grafana/nanogit/commit/41a17974ba1c7cce4c5bee52bc69800257334a3d))


### Documentation

* Update CHANGELOG for initial release preparation ([#114](https://github.com/grafana/nanogit/issues/114)) ([6dd7a24](https://github.com/grafana/nanogit/commit/6dd7a24c9b88deca40f296ea91d160b5783ad2fe))

## [0.1.0](https://github.com/grafana/nanogit/compare/v0.0.0...v0.1.0) (2025-11-11)


### Features

* add automated release pipeline with semantic versioning ([#106](https://github.com/grafana/nanogit/issues/106)) ([3c7f85a](https://github.com/grafana/nanogit/commit/3c7f85a9ace595ff8211f99b04a3700ffdc4f6f8))
* implement CHANGELOG updates via auto-merge PRs ([#108](https://github.com/grafana/nanogit/issues/108)) ([45c6be7](https://github.com/grafana/nanogit/commit/45c6be75f43f6ac4b650492a227434ff43f26dc6))


### Bug Fixes

* resolve YAML syntax error in release workflow ([#109](https://github.com/grafana/nanogit/issues/109)) ([4669671](https://github.com/grafana/nanogit/commit/46696718e4efd42f9096bbc7c63c5a47f6a971c7))
* use correct commit SHA for wait-on-check-action ([#107](https://github.com/grafana/nanogit/issues/107)) ([7f97690](https://github.com/grafana/nanogit/commit/7f976905a288d1c16a5990ad6da7ba6eaeb86539))


### Documentation

* document v0.x.x initial version strategy ([#111](https://github.com/grafana/nanogit/issues/111)) ([6e88621](https://github.com/grafana/nanogit/commit/6e886217dc058330715bae3972b49b5c46712d85))

### Notable Features

- HTTPS-only Git operations with Smart HTTP Protocol v2
- Stateless architecture with no local .git directory dependency
- Memory-optimized design with streaming packfile operations
- Flexible storage architecture with pluggable object storage
- Cloud-native authentication (Basic Auth and API tokens)
- Essential Git operations (read/write objects, commit operations, diffing)
- Path filtering with glob patterns for clones
- Configurable writing modes (memory/disk/auto)
- Batch blob fetching and concurrent operations for improved performance

---

**Note:** This changelog is automatically generated. Manual edits will be overwritten on the next release.
