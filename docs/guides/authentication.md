# Authentication

nanogit is HTTPS-only and token-oriented: there is no SSH key management. Two options cover every provider, both passed to `NewHTTPClient`.

## Basic auth (most providers)

`options.WithBasicAuth(username, password)` sends a standard HTTP basic-auth header. With personal access tokens, the username is usually ignored by the provider — `"git"` is a safe convention:

```go
client, err := nanogit.NewHTTPClient(
    "https://github.com/owner/repo.git",
    options.WithBasicAuth("git", token),
)
```

Provider conventions for the username/password pair:

| Provider | Username | Password |
| -------- | -------- | -------- |
| GitHub | anything (`git`) | personal access token (classic or fine-grained with `contents` read/write) |
| GitLab | `oauth2` | personal access token with `read_repository`/`write_repository` |
| Bitbucket Cloud | your username | app password with repository read/write |
| Gitea / Forgejo | anything (`git`) | access token |

## Raw Authorization header

`options.WithTokenAuth(token)` sets the `Authorization` header verbatim — nanogit adds no prefix, so include the scheme yourself. Use this for OAuth2 bearer tokens, GitHub App installation tokens, or any server with a custom scheme:

```go
client, err := nanogit.NewHTTPClient(
    "https://gitlab.com/owner/repo.git",
    options.WithTokenAuth("Bearer "+accessToken),
)
```

## No authentication

Public repositories need no credentials for reads:

```go
client, err := nanogit.NewHTTPClient("https://github.com/grafana/nanogit.git")
```

## Probing what your credentials can do

Before wiring a repository into a service, probe access explicitly:

```go
canRead, err := client.CanRead(ctx)   // git-upload-pack reachable?
canWrite, err := client.CanWrite(ctx) // git-receive-pack reachable?
```

`CanWrite` checks **repository-level** write access only — it cannot see branch protection rules, so a push to a protected branch can still be rejected. Authentication failures surface as `nanogit.ErrUnauthorized` (bad or missing credentials) or `nanogit.ErrPermissionDenied` (valid credentials, insufficient rights); see [Error handling](error-handling.md).

## Credential hygiene

- Pass tokens from your secret store at client construction. Credentials travel only in the `Authorization` header; nanogit's own [logging](https://pkg.go.dev/github.com/grafana/nanogit/log) never includes them.
- Create one client per repository/credential pair. Clients are cheap: no clone, no per-repository state.
- The CLI takes the same credentials via `--username`/`--token` flags or `NANOGIT_USERNAME`/`NANOGIT_TOKEN` env vars — see the [CLI docs](../getting-started/cli.md#authentication).
