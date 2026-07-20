# Architecture Overview

nanogit is designed as a lightweight, cloud-native Git implementation with a focus on stateless operations and minimal resource usage.

## Core Design Principles

### Stateless Operations
nanogit operates without requiring a local .git directory, making it ideal for serverless functions, containers, and microservices where persistent local state isn't available or desired.

### HTTPS-Only Protocol
Focuses exclusively on Git Smart HTTP Protocol v2, eliminating the complexity of supporting multiple transport protocols and simplifying authentication in cloud environments.

#### Why protocol v2 only (and not v1)?
nanogit implements Git Smart HTTP Protocol **v2** and intentionally does not implement the legacy v1 protocol. `IsServerCompatible` (see `protocol/client/compatibility.go`) detects the server's protocol version up front and treats a v1-only server as incompatible rather than falling back. The reasons are architectural, not incidental:

- **Statelessness.** v2 replaced v1's stateful, multi-round `want`/`have` negotiation with a command-oriented model that completes in a single stateless HTTP round trip. That matches nanogit's stateless design (no local `.git`, no persistent connection state); v1's negotiation assumes connection state nanogit deliberately does not keep.
- **Server-side ref filtering.** v2's `ls-refs` command lets the client ask for only the refs it needs via `ref-prefix`. v1 advertises *every* ref on each `info/refs` request, which does not scale for the large, multitenant repositories nanogit targets.
- **Minimal surface area.** Supporting both wire protocols would double the negotiation and parsing paths. Keeping to v2 alone honors the minimal-surface-area principle below — less code, fewer bugs, smaller attack surface.
- **Broad, though not universal, reach.** Protocol v2 has shipped since Git 2.18 (2018) and been the default fetch protocol since Git 2.26 (2020). GitHub, GitLab, and Bitbucket support it. The known exception is **Azure DevOps / Azure Repos**, which only advertises protocol v1; nanogit treats it as incompatible and does not fall back. For nanogit's intended audience the trade-off is deliberate — v1 support would add complexity to reach a shrinking set of servers.

### Pluggable Storage
Features a flexible two-layer storage architecture:
- **Writing modes**: Control temporary storage during packfile creation
- **Object storage**: Handle long-term caching with pluggable backends

### Retry Mechanism
Pluggable retry mechanism for robust operations:
- **Context-based injection**: Retriers configured via Go context
- **Built-in exponential backoff**: Configurable retry with jitter
- **Custom retriers**: Implement custom retry logic via interface
- **Backward compatible**: Default behavior is no retries

### Minimal Surface Area
Implements only essential Git operations, reducing attack surface and resource requirements compared to full Git implementations.

## Key Components

### Client Interface
The primary interface for Git read operations:
- Reference resolution (branches, tags)
- Object retrieval (blobs, commits, trees)
- Repository cloning with path filtering
- Commit comparison and diffing

### StagedWriter Interface
Transactional interface for batched write operations:
- File creation, updates, and deletions
- Staged operations for atomic commits
- Configurable writing modes (memory/disk/auto)
- Push operations to remote repositories

### Protocol Layer
Implementation of Git Smart HTTP Protocol:
- Git protocol v2 capabilities
- Packfile processing (blobs, commits, trees, deltas)
- Reference advertisement and negotiation
- Authentication handling (Basic Auth, tokens)

### Storage System
Context-based pluggable storage backends:
- Default in-memory implementation
- Custom storage via dependency injection
- Separation of temporary and persistent storage
- Optimized for cloud-native patterns

## Architecture Diagrams

### Read Operation Flow
```
Client Request
    ↓
HTTP Protocol Layer
    ↓
Object Storage (check cache)
    ↓
Remote Git Server (if cache miss)
    ↓
Packfile Processing
    ↓
Object Decompression
    ↓
Return to Client
```

### Write Operation Flow
```
Staged Writer
    ↓
Writing Mode Selection
    ↓
Object Creation (memory/disk)
    ↓
Packfile Generation
    ↓
HTTP Protocol Layer
    ↓
Remote Git Server
    ↓
Push Confirmation
```

## Related Documentation

- [Storage Architecture](storage.md) - Detailed storage backend design
- [Retry Mechanism](retry.md) - Pluggable retry mechanism for robust operations
- [Delta Resolution](delta-resolution.md) - Git delta handling implementation
- [Performance](performance.md) - Performance characteristics and benchmarks
