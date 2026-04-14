# CLAUDE.md — docker/

## Overview

Centralized Docker configuration for the Go workspace (ADR-016). One subdirectory per service, each containing a `Dockerfile`.

## Structure

```
docker/
├── api/
│   └── Dockerfile      # multi-stage build for the HTTP API binary
├── utils/
│   └── Dockerfile      # multi-stage build for the migration runner binary
```

Related files at workspace root:
- `.dockerignore` — controls what gets sent to the Docker daemon
- `docker-compose.yaml` — orchestrates the full local stack (issue #99)

## Build Context

The build context is **always the workspace root**, not this directory. The Go workspace (`go.work`) and all three modules (`api/`, `utils/`, `internal/shared/`) must be accessible during the build.

```bash
# from workspace root
docker build -f docker/api/Dockerfile .
docker build -f docker/utils/Dockerfile .
```

All `COPY` paths inside Dockerfiles are relative to the workspace root, not the Dockerfile location.

## Conventions

- **Base images:** `golang:1.26-alpine` (builder), `alpine:latest` (runtime)
- **Multi-stage:** builder compiles the binary; runtime stage copies only the binary — no Go toolchain or source in the final image
- **Non-root user:** runtime stage creates `appuser`/`appgroup` via `adduser -S` / `addgroup -S` and runs as `appuser`
- **Config:** all configuration via environment variables at runtime — no config files baked into images
- **`EXPOSE`:** each Dockerfile declares the port the service listens on (e.g. `EXPOSE 8080` for api)
- **`utils`:** runs migrations and exits (not a long-running service); `//go:embed configs/config.json` requires the file to exist at build time (even as `{}`)

## go.work Stub Pattern

`go.work` references all three modules. When building a single service (e.g. `api`), the other modules' source code isn't needed — but `go.work` still expects their `go.mod` to exist for dependency resolution. Each Dockerfile copies only the `go.mod` from sibling modules and stubs any missing ones:

```dockerfile
COPY utils/go.mod utils/go.sum* ./utils/
RUN mkdir -p utils && test -f utils/go.mod || echo 'module commerce/utils' > utils/go.mod
```

This avoids copying unnecessary source while keeping `go mod download` functional. Only the target service's source is copied after the dependency cache layer.

## Dockerfile Layering Strategy

1. Copy `go.work` + all `go.mod`/`go.sum` files
2. Stub sibling modules if needed
3. `go mod download` — cached unless dependencies change
4. Copy only the target service's source + `internal/shared/`
5. Build the binary
6. Runtime stage copies only the binary
