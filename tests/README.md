# nanogit integration tests

This directory is a separate Go module (`github.com/grafana/nanogit/tests`) containing the integration and provider test suites for nanogit. It is **internal**: it is never tagged or published, and it resolves the `nanogit` and `gittest` modules via local `replace` directives (and the repo's `go.work`).

## What's here

- **Integration tests** (`*_integration_test.go`, `integration_suite_test.go`): a [Ginkgo](https://onsi.github.io/ginkgo/)/[Gomega](https://onsi.github.io/gomega/) suite that runs against a containerized [Gitea](https://gitea.io/) server, started via the public [`gittest`](../gittest/README.md) package (Testcontainers-based). The container is shared across the suite and cleaned up automatically.
- **Provider compatibility tests** (`providers_integration_test.go`, `sign_providers_verify_test.go`, `provider_limits_integration_test.go`): plain `go test` tests that exercise real Git hosting providers (GitHub, GitLab, Bitbucket) end to end, including commit-signing verification.

## Running the tests

Run everything from the **repository root**.

### Integration tests (requires Docker)

```bash
make test-integration
```

This runs Ginkgo with `-focus "Integration"` against `./tests` (race detector on, randomized, parallel). Docker must be running so the Gitea container can start.

### Provider tests (requires credentials)

```bash
export TEST_REPO=https://github.com/grafana/nanogit-test.git  # URL of a test repository
export TEST_USER=git                                          # username for basic auth
export TEST_TOKEN=<token>                                     # token with write access
make test-providers
```

This runs `go test -race -run 'TestProviders|TestSignProvidersVerify' ./tests`. Both tests **skip** unless `TEST_REPO`, `TEST_USER`, and `TEST_TOKEN` are all set. Note the tests create and delete branches and push commits, so point them at a dedicated test repository.

`TestSignProvidersVerify` additionally skips unless `TEST_SIGN_EMAIL` is set, and uses these optional variables:

- `TEST_PROVIDER` — `github`, `gitlab`, or `bitbucket` (selects the signature-verification API)
- `TEST_GPG_KEY`, `TEST_SSH_KEY`, `TEST_SMIME_KEY`, `TEST_SMIME_CERT` — signing key material to test with
- `TEST_CLEANUP` — set to `false` to keep the test branches around for debugging

CI runs the provider tests against GitHub, GitLab, and Bitbucket test repositories; see `.github/workflows/ci.yml` and [CONTRIBUTING.md](../CONTRIBUTING.md) for details.
