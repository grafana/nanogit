# Why nanogit exists

nanogit was created by Grafana to power [Git Sync](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/git-sync/) — the [Observability as Code](https://grafana.com/docs/grafana/latest/as-code/) feature that lets you store Grafana dashboards and folders as files in your own Git repository and manage them as code: auditable, reviewable, and reproducible, editing in the UI and opening pull requests without leaving Grafana.

This page tells the story behind the library: the problem it was built to solve, the constraints that shaped its design, and where it runs in production today. If you just want to know what nanogit does and how to use it, start with the [overview](index.md) and the [Quick Start](getting-started/quick-start.md).

## The problem

Git Sync needed a real Git engine embedded in Grafana's backend, and none of the obvious options fit:

- **A GitHub API wasn't enough.** The goal grew from "sync with GitHub" to "sync with any Git provider." That means speaking the standard Git Smart HTTP protocol directly, so GitLab, Bitbucket, and self-hosted servers work without provider-specific code.
- **The `git` CLI and libgit2** assume a local `.git` working directory. Maintaining a checkout per tenant, and shelling out to a binary, is impractical and hard to reason about operationally in a shared, multitenant backend.
- **[go-git](https://github.com/go-git/go-git)** is mature, but its abstraction and overhead were too heavy for this use case. It is built around local-disk operations and full clones — stateful and memory-heavy once multiplied across thousands of tenants and large repositories.

## The constraints that shaped the design

So we built nanogit around the constraints Git Sync actually has:

- **Provider-agnostic** — talks the Git wire protocol over HTTPS, so any compliant provider works without bespoke integrations.
- **Stateless — no clones, no `.git`** — a defining goal was to avoid full clones and never persist a `.git` directory or per-tenant working tree. nanogit reads and writes objects directly over HTTPS, so there is no local repository state to store, clean up, or keep consistent across a horizontally scaled, multitenant backend.
- **Lean and fast** — parses only what Git Sync needs. Streaming packfiles, path filtering, shallow reads, and delta handling keep it fast and memory-efficient on Grafana-scale repositories, where an operation usually touches a subpath rather than the whole repo.
- **Operational control** — speaking the Git protocol directly lets Grafana control exactly how it talks to each provider: how many requests it makes and how large each response is (to stay within provider rate limits and byte budgets), what objects get cached and reused across operations, and how many round trips a sync costs. Hosted provider APIs and general-purpose clients don't expose that level of control.
- **Safe and controllable** — a focused, embeddable library with a minimal surface area is easier to secure and operate than a general-purpose tool, and it needs only token auth (no SSH key management).

nanogit is open source and usable on its own, but its design — and its [performance characteristics](architecture/performance.md) — come directly from this workload: doing the Git plumbing behind Git Sync efficiently, safely, and for many tenants at scale.

## Who uses nanogit

- **[grafana/grafana](https://github.com/grafana/grafana)** — nanogit is the Git engine behind [Git Sync](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/git-sync/) provisioning (`apps/provisioning/pkg/repository/git`). It resolves refs, reads and writes dashboards and folders, stages commits, and pushes to each tenant's repository over HTTPS.
- **[grafana/grafana-bench](https://github.com/grafana/grafana-bench)** — nanogit is the **default Git driver** for fetching the test-suite repository before a benchmark run. grafana-bench pulls its k6/Playwright suites from Git; nanogit's lightweight, HTTP-only client with parallel object fetching and sparse/subpath checkout pulls large monorepos faster and with less overhead than the alternative go-git driver, keeping benchmark setup fast and repeatable. (go-git stays available via `--git-driver gogit` when SSH or full Git features are needed.)

Using nanogit in your project? [Open an issue or PR](https://github.com/grafana/nanogit/issues) and we'll add you here.

## Learn more

- [Overview](index.md) — what nanogit is, when to use it, and how it compares to go-git
- [Architecture Overview](architecture/overview.md) — design principles, including [why protocol v2 only](architecture/protocol-v2.md)
- [Performance](architecture/performance.md) — benchmark methodology and results against go-git and the git CLI
