# Why Git Protocol v2 Only?

nanogit speaks **only** [Git Smart HTTP Protocol v2](https://git-scm.com/docs/protocol-v2). It does not implement the legacy v1 protocol (neither the original "smart" v1 negotiation nor the "dumb" HTTP protocol), and it will not silently fall back to it. This is a deliberate design decision, not a missing feature:

- **Stateless by design** — Protocol v2 replaces v1's stateful, multi-round `want`/`have` negotiation with a command-oriented request model that completes in a single stateless HTTP round trip. That maps directly onto nanogit's stateless, serverless-friendly architecture. v1's negotiation assumes connection state that nanogit deliberately does not keep.
- **Server-side ref filtering** — v2's `ls-refs` command lets the client request only the references it needs (via `ref-prefix`). v1 dumps the *entire* ref advertisement on every `info/refs` request, which is wasteful for repositories with thousands of branches and tags — exactly the multitenant, large-repo case nanogit targets.
- **Smaller surface area** — Supporting both protocols would double the negotiation and parsing code paths. Minimal surface area is a core design principle: less code means fewer bugs and a smaller attack surface.
- **Broad (not universal) provider support** — Protocol v2 has been available since Git 2.18 (2018) and the default fetch protocol since Git 2.26 (2020). GitHub, GitLab, and Bitbucket all support it. The notable exception is **Azure DevOps / Azure Repos**, which only speaks protocol v1 — nanogit cannot talk to it, and there is no fallback. For nanogit's target audience (cloud-native services talking to modern hosted Git), the trade-off is worth it: v1 support would add complexity to serve a shrinking set of servers.

## Fail fast, not degrade

When a server only speaks v1, nanogit fails fast rather than silently degrading:

- In code, `IsServerCompatible` (see `protocol/client/compatibility.go`) detects the server's protocol version up front and treats a v1-only server as incompatible.
- From the terminal, [`nanogit check`](../getting-started/server-compatibility.md) reports the server as incompatible, and every operation fails with a clear error.

Always run the check against a new provider before integrating. See the [Server Compatibility guide](../getting-started/server-compatibility.md) for how to verify support.

## Related

- [Architecture Overview](overview.md) — the design principles this decision serves
- [Learn how Git works](../how-git-works.md) — the upstream protocol documentation, including the v2 specification
