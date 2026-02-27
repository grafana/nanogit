# Installation

## Prerequisites

- **Go 1.25 or later** - nanogit requires Go 1.25+ for building and using the library
- **Git** (for development) - Only needed if you're contributing to nanogit

## Installing nanogit

### Latest Version

To install the latest version of nanogit:

```bash
go get github.com/grafana/nanogit@latest
```

### Specific Version

To install a specific version:

```bash
go get github.com/grafana/nanogit@v0.x.x
```

Replace `v0.x.x` with the version you want to install. See all available versions on the [releases page](https://github.com/grafana/nanogit/releases).

## Verifying Installation

After installation, verify that nanogit is available in your Go module:

```bash
go list -m github.com/grafana/nanogit
```

This should display the installed version.

## Next Steps

- [Quick Start Guide](quick-start.md) - Get started with basic usage
- [API Reference (GoDoc)](https://pkg.go.dev/github.com/grafana/nanogit) - Explore the full API
- [Examples](https://github.com/grafana/nanogit/tree/main/examples) - See working code examples
