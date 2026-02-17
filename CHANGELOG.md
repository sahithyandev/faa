# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Version command to display version, commit, build date, and platform information
- Versioning support with build-time injection using ldflags
- CHANGELOG.md for tracking changes
- Release pipeline with GitHub Actions and GoReleaser
- Binary distribution via GitHub Container Registry (GHCR)
- Support for macOS and Linux (amd64 and arm64) platforms

## [0.1.0] - TBD

### Added
- Initial release of faa (localhost-dev)
- Development server manager with stable HTTPS URLs
- Background daemon for managing development servers
- Stable port assignment based on project name
- HTTPS access via .local domains
- Embedded Caddy reverse proxy with automatic certificate generation
- Support for Node.js projects with package.json
- Commands: setup, daemon, run, status, stop, routes, ca-path, clean
- macOS LaunchDaemon support
- Linux support with setcap for privileged port binding
- Automatic CA certificate installation

[Unreleased]: https://github.com/sahithyandev/faa/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/sahithyandev/faa/releases/tag/v0.1.0
