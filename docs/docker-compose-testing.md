# Docker Compose Testing Environment

This document describes how to use the Docker Compose setup for testing nanogit's Git protocol v1 detection feature.

## Overview

The Docker Compose environment provides:
- **Gitea server** configured to ONLY support Git protocol v1 (protocol v2 disabled)
- **Git client container** with Git 2.15 (pre-protocol-v2) for manual testing

This setup is designed for manual testing and validation. Integration tests continue to use testcontainers-go.

## Prerequisites

- Docker and Docker Compose installed
- Repository cloned locally

## Quick Start

### 1. Start the Environment

```bash
docker-compose up -d
```

This will:
- Start Gitea server on http://localhost:3000
- Start a Git client container with Git 2.15
- Wait for Gitea to be healthy before starting the client

### 2. Verify Gitea is Running

```bash
# Check service health
docker-compose ps

# Verify Gitea version
curl http://localhost:3000/api/v1/version
```

### 3. Access Gitea Web UI

Open http://localhost:3000 in your browser.

**Admin Credentials:**
- Username: `giteaadmin`
- Password: `admin123`

## Creating Test Repositories

### Option 1: Via Web UI

1. Log in to http://localhost:3000
2. Click "+" → "New Repository"
3. Create a test repository with some initial content

### Option 2: Via API

```bash
# Create a repository
curl -X POST "http://localhost:3000/api/v1/user/repos" \
  -u "giteaadmin:admin123" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-v1",
    "description": "Test repository for protocol v1",
    "auto_init": true,
    "default_branch": "main"
  }'
```

### Option 3: Via Git Client Container

```bash
# Exec into the git client container
docker-compose exec git-client-v1 sh

# Configure git
git config --global user.name "Test User"
git config --global user.email "test@example.com"

# Create a test repo
mkdir test-repo && cd test-repo
git init
echo "# Test Repo" > README.md
git add README.md
git commit -m "Initial commit"

# Push to Gitea (create repo via UI first)
git remote add origin http://giteaadmin:admin123@gitea-v1:3000/giteaadmin/test-repo-v1.git
git push -u origin main
```

## Testing Protocol v1

### Verify Protocol v1 is Used

```bash
# Exec into the git client container
docker-compose exec git-client-v1 sh

# Clone with packet tracing to verify protocol v1
GIT_TRACE_PACKET=1 git clone http://giteaadmin:admin123@gitea-v1:3000/giteaadmin/test-repo-v1.git

# Look for packet traces - you should NOT see "version=2" in the output
# Protocol v1 uses standard git-upload-pack without version negotiation
```

### What to Look For

**Protocol v1 (expected):**
```
packet: git< # service=git-upload-pack
packet: git< 0000
```

**Protocol v2 (should NOT appear):**
```
version 2
```

### Verify Git Version

```bash
docker-compose exec git-client-v1 git --version
# Expected: git version 2.15.x (pre-protocol-v2)
```

## Testing with nanogit

### From Host Machine

```bash
# Build nanogit
go build ./cmd/nanogit

# Test protocol detection
./nanogit detect-protocol http://giteaadmin:admin123@localhost:3000/giteaadmin/test-repo-v1.git

# Or test your application code that uses nanogit
go test ./tests -run TestProtocolDetection
```

### From Git Client Container

```bash
docker-compose exec git-client-v1 sh

# Install Go in the container (if needed for testing)
apk add --no-cache go

# Run tests
cd /workspace
go test ./protocol/client -v
```

## Viewing Logs

```bash
# View Gitea logs
docker-compose logs -f gitea-v1

# View all logs
docker-compose logs -f
```

## Stopping the Environment

```bash
# Stop services (keeps data)
docker-compose stop

# Stop and remove containers (keeps volumes)
docker-compose down

# Stop, remove containers and volumes (full cleanup)
docker-compose down -v
```

## Troubleshooting

### Gitea Won't Start

```bash
# Check logs
docker-compose logs gitea-v1

# Verify port 3000 is not in use
lsof -i :3000

# Remove volumes and restart
docker-compose down -v
docker-compose up -d
```

### Can't Connect from Git Client

```bash
# Verify network connectivity
docker-compose exec git-client-v1 ping gitea-v1

# Verify Gitea is healthy
docker-compose exec git-client-v1 curl http://gitea-v1:3000/api/v1/version
```

### Git Operations Fail

```bash
# Check credentials
curl -u "giteaadmin:admin123" http://localhost:3000/api/v1/user

# Verify repository exists
curl http://localhost:3000/api/v1/repos/giteaadmin/test-repo-v1
```

## Configuration Details

### Gitea Environment Variables

The key configuration for protocol v1:
- `GITEA__repository__ENABLE_AUTO_GIT_WIRE_PROTOCOL=false` - Disables Git protocol v2

All other environment variables match the integration test setup in `gittest/server.go`.

### Git Version Matrix

| Git Version | Protocol v2 Support | Used In |
|-------------|---------------------|---------|
| < 2.18      | No (v1 only)        | git-client-v1 (Alpine 3.7 = Git 2.15) |
| >= 2.18     | Yes (v2 default)    | Modern Git installations |

### Network Architecture

```
┌─────────────────┐
│   Host Machine  │
│   (localhost)   │
└────────┬────────┘
         │ port 3000
         ▼
┌─────────────────┐     ┌──────────────────┐
│   gitea-v1      │◄────┤ git-client-v1    │
│   (protocol v1) │     │ (Git 2.15)       │
└─────────────────┘     └──────────────────┘
         │                       │
         └───────────────────────┘
           nanogit-test network
```

## Advanced Usage

### Testing with Network Latency

Add latency simulation using `tc` (traffic control):

```bash
docker-compose exec git-client-v1 sh -c "apk add --no-cache iproute2"
docker-compose exec git-client-v1 tc qdisc add dev eth0 root netem delay 100ms
```

### Testing with Different Gitea Versions

Edit `docker-compose.yml` to change the image tag:
```yaml
image: gitea/gitea:1.20  # or another version
```

### Adding a Modern Git Client for Comparison

Add to `docker-compose.yml`:
```yaml
  git-client-v2:
    image: alpine:latest
    container_name: nanogit-git-client-v2
    command: tail -f /dev/null
    volumes:
      - ./:/workspace
    working_dir: /workspace
    networks:
      - nanogit-test
    entrypoint: /bin/sh -c "apk add --no-cache git curl && tail -f /dev/null"
```

Then test protocol v2:
```bash
docker-compose exec git-client-v2 git --version  # Should be >= 2.18
GIT_TRACE_PACKET=1 git clone http://giteaadmin:admin123@gitea-v1:3000/giteaadmin/test-repo.git
# Note: Will fall back to v1 since server doesn't support v2
```

## Integration with Makefile

You can add these targets to the `Makefile`:

```makefile
.PHONY: docker-up docker-down docker-logs docker-test-v1

docker-up:
	docker-compose up -d
	@echo "Waiting for Gitea to be ready..."
	@sleep 5
	@echo "Gitea available at http://localhost:3000"

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-test-v1:
	@echo "Testing protocol v1 detection..."
	docker-compose exec git-client-v1 sh -c "cd /workspace && go test ./protocol/client -v"
```

## References

- [Git Protocol v2 Documentation](https://git-scm.com/docs/protocol-v2)
- [Gitea Configuration Cheat Sheet](https://docs.gitea.com/administration/config-cheat-sheet)
- Nanogit integration tests: `tests/providers_integration_test.go`
- Gitea test server setup: `gittest/server.go`
