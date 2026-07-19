# Learn how Git works

nanogit implements the Git Smart HTTP protocol from scratch, so the upstream Git documentation is the best companion to its source code — it describes exactly what happens on the wire when nanogit talks to a server.

## Resources

Want to learn how Git works? The following resources are useful:

- [Git on the Server - The Protocols](https://git-scm.com/book/ms/v2/Git-on-the-Server-The-Protocols)
- [Git Protocol v2](https://git-scm.com/docs/protocol-v2)
- [Pack Protocol](https://git-scm.com/docs/pack-protocol)
- [Git HTTP Backend](https://git-scm.com/docs/git-http-backend)
- [HTTP Protocol](https://git-scm.com/docs/http-protocol)
- [Git Protocol HTTP](https://git-scm.com/docs/gitprotocol-http)
- [Git Protocol v2](https://git-scm.com/docs/gitprotocol-v2)
- [Git Protocol Pack](https://git-scm.com/docs/gitprotocol-pack)
- [Git Protocol Common](https://git-scm.com/docs/gitprotocol-common)

## Related

- [Architecture Overview](architecture/overview.md) — how nanogit maps these protocols onto a stateless client, including [why protocol v2 only](architecture/overview.md#why-protocol-v2-only-and-not-v1)
- [Delta Resolution](architecture/delta-resolution.md) — how nanogit resolves packfile deltas
