<div style="text-align: center; margin-bottom: 2rem;">
  <img src="/banner.png" alt="nanogit - Git reimagined for the cloud – in Go" style="max-width: 100%; height: auto;">
</div>

<p style="display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; justify-content: center; margin-bottom: 2rem;">
  <a href="https://github.com/grafana/nanogit/releases"><img src="https://img.shields.io/github/v/release/grafana/nanogit" alt="GitHub Release"></a>
  <a href="https://github.com/grafana/nanogit/stargazers"><img src="https://img.shields.io/github/stars/grafana/nanogit?style=social" alt="GitHub Stars"></a>
  <a href="https://github.com/grafana/nanogit/blob/main/LICENSE.md"><img src="https://img.shields.io/github/license/grafana/nanogit" alt="License"></a>
  <a href="https://github.com/grafana/nanogit/actions/workflows/ci.yml"><img src="https://github.com/grafana/nanogit/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/grafana/nanogit"><img src="https://pkg.go.dev/badge/github.com/grafana/nanogit.svg" alt="Go Reference"></a>
  <a href="https://codecov.io/gh/grafana/nanogit"><img src="https://codecov.io/gh/grafana/nanogit/branch/main/graph/badge.svg" alt="codecov"></a>
</p>

## What is nanogit?

nanogit is a lightweight Git client library for Go, built for services that read from and write to Git repositories over HTTPS — with no local clone, no `.git` directory, and no `git` binary. It speaks the [Git Smart HTTP Protocol v2](https://git-scm.com/docs/protocol-v2) directly, so it works with GitHub, GitLab, Bitbucket, Gitea, and any other server that supports protocol v2.

Grafana built nanogit to power [Git Sync](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/git-sync/), which syncs dashboards with tenants' own Git repositories from inside Grafana's multitenant backend — a workload where cloning every repository to disk is not an option. Read the full story in [Why nanogit exists](why-nanogit.md).

- **Stateless** — reads and writes Git objects directly over HTTPS; nothing is persisted locally, so there is no per-repository state to store, clean up, or keep consistent across replicas
- **Works with any protocol v2 server** — one code path for GitHub, GitLab, Bitbucket, Gitea, and self-hosted servers; token-based auth, no SSH key management
- **Essential operations** — refs, blobs, trees, commits, diffs, staged writes, and shallow clones with glob-based path filtering
- **Memory-efficient** — streaming packfile processing and configurable memory/disk/auto writing modes for bulk operations
- **Fast** — orders of magnitude faster and leaner than a full Git implementation for common server-side operations ([benchmarks below](#how-is-it-different-from-go-git))
- **Commit signing** — sign commits with GPG, SSH, or S/MIME keys
- **Pluggable** — object storage (caching) and [retry policies](architecture/retry.md) are injected via context, with sensible defaults

## When should I use it?

Use nanogit when your code runs **server-side and talks to Git over HTTPS**:

- **GitOps and as-code services** — sync configuration, dashboards, or manifests between your application and users' repositories
- **Bots and automation** — commit generated files, open changes, or mirror content without shelling out to `git`
- **Multitenant platforms** — operate on thousands of repositories without maintaining a checkout per tenant
- **Serverless and containers** — environments with little or no persistent disk
- **CI tooling** — fetch only the subpaths you need from large repositories using path-filtered, shallow clones

## When should I not use it?

nanogit is deliberately narrow. Reach for the `git` CLI or [go-git](https://github.com/go-git/go-git) instead when you need:

- **Local development workflows** — working trees, the index, `.git` directories, or repositories on disk
- **Full Git functionality** — merges, rebases, blame, hooks, or Git configuration management
- **Other transports** — SSH, `git://`, or local file access; nanogit is HTTPS-only
- **Protocol v1 or "dumb" HTTP servers** — nanogit requires Smart HTTP protocol v2 and does not fall back. Notably, **Azure DevOps / Azure Repos only speaks v1 and is not supported.** Run [`nanogit check`](getting-started/server-compatibility.md) against a new provider before integrating
- **Signature verification** — nanogit can sign commits but does not verify signatures
- **Fine-grained file permissions** — all files are written with mode 0644

See [Why Git Protocol v2 Only?](architecture/protocol-v2.md) for the rationale behind the strictest of these constraints.

## How is it different from go-git?

[go-git](https://github.com/go-git/go-git) is a mature, full-featured Git implementation. nanogit trades that breadth for a small, stateless core optimized for cloud services:

| Feature        | nanogit                                                 | go-git                 |
| -------------- | ------------------------------------------------------- | ---------------------- |
| Protocol       | HTTPS only (Smart HTTP v2)                              | All protocols          |
| Storage        | Stateless; pluggable object storage and writing modes   | Local disk operations  |
| Cloning        | Shallow, with glob-based path filtering                 | Full repository clones |
| Scope          | Essential operations only                               | Full Git functionality |
| Use case       | Cloud services, multitenant backends                    | General purpose        |
| Resource usage | Minimal footprint                                       | Full Git features      |

Because it never materializes a full repository, nanogit is dramatically faster and lighter for typical server-side operations. From the July 2025 run of the [benchmark suite](https://github.com/grafana/nanogit/tree/main/perf), on the XL repository tier (15,000 files, 3,000 commits), nanogit vs go-git:

| Scenario             | Speed         | Memory usage |
| -------------------- | ------------- | ------------ |
| CreateFile (XL repo) | 281.6x faster | 198.4x less  |
| UpdateFile (XL repo) | 297.3x faster | 189.2x less  |
| DeleteFile (XL repo) | 280.5x faster | 200.5x less  |
| GetFlatTree (XL repo) | 260.8x faster | 154.3x less  |

(go-git did not complete the bulk-create and commit-comparison scenarios in that run, so no multipliers are quoted for them; nanogit bulk-created 1,000 files in ~103ms.) See the [performance analysis](architecture/performance.md) for methodology and complete results.

## Is it production-ready?

**Yes.** nanogit is the Git engine behind [Git Sync](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/git-sync/) in [grafana/grafana](https://github.com/grafana/grafana), reading and writing dashboards across tenants' repositories in production, and the default Git driver in [grafana-bench](https://github.com/grafana/grafana-bench). See [who uses nanogit](why-nanogit.md#who-uses-nanogit).

Releases follow [semantic versioning](https://semver.org/): the v1 API is stable, and breaking changes only land in major versions. The project is actively developed by Grafana Labs.

## Getting started

Install the library (requires **Go 1.26+**):

```bash
go get github.com/grafana/nanogit@latest
```

Then follow the guides:

- **[Installation](getting-started/installation.md)** — install nanogit in your project
- **[Quick Start](getting-started/quick-start.md)** — read, write, clone, retry, and authenticate in a few minutes
- **[CLI](getting-started/cli.md)** — terminal-based Git operations for testing and demos
- **[Server Compatibility](getting-started/server-compatibility.md)** — verify your Git server supports nanogit in four CLI commands
- **[API Reference (GoDoc)](https://pkg.go.dev/github.com/grafana/nanogit)** — complete API documentation

## Guides

Task-focused guides for production use:

- **[Writing with the StagedWriter](guides/writing.md)** — the transactional write model: staging, multi-commit, push, retry semantics
- **[Authentication](guides/authentication.md)** — basic auth, raw tokens, and per-provider conventions
- **[Error Handling](guides/error-handling.md)** — sentinel and typed errors, `errors.Is`/`errors.As` patterns
- **[Commit Signing](guides/commit-signing.md)** — GPG, SSH, and S/MIME signatures
- **[Response Limits](guides/response-limits.md)** — cap response sizes for multitenant safety
- **[History and Diffs](guides/history.md)** — `ListCommits` pagination/filtering and `CompareCommits`

## Architecture

Learn about nanogit's design and internals:

- **[Architecture Overview](architecture/overview.md)** — core design principles and components
- **[Storage Backend](architecture/storage.md)** — pluggable storage and writing modes
- **[Retry Mechanism](architecture/retry.md)** — pluggable retry mechanism for robust operations
- **[Delta Resolution](architecture/delta-resolution.md)** — Git delta handling implementation
- **[Performance](architecture/performance.md)** — performance characteristics and benchmarks
- **[Learn how Git works](how-git-works.md)** — pointers to the upstream Git protocol documentation nanogit implements

## Testing

nanogit ships the tooling to test code that depends on it:

- **[Testing Guide](testing-guide.md)** — complete guide with patterns and best practices
- **[gittest Package](https://pkg.go.dev/github.com/grafana/nanogit/gittest)** — integration testing with a real containerized Git server, local repository helpers, and automatic cleanup
- **Unit testing** — generated mocks for the `Client` and `StagedWriter` interfaces

## Contributing

We welcome contributions! Please see the [Contributing Guide](https://github.com/grafana/nanogit/blob/main/CONTRIBUTING.md) for details on how to submit pull requests, report issues, and set up your development environment. This project follows the [Grafana Code of Conduct](https://github.com/grafana/nanogit/blob/main/CODE_OF_CONDUCT.md).

## License

This project is licensed under the [Apache License 2.0](https://github.com/grafana/nanogit/blob/main/LICENSE.md).

## Security

If you find a security vulnerability, please report it according to [our security policy](https://github.com/grafana/.github/blob/main/SECURITY.md).

## Support

- GitHub Issues: [Create an issue](https://github.com/grafana/nanogit/issues)
- Community: [Grafana Community Forums](https://community.grafana.com)
