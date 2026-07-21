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

**No `docker/relay/Dockerfile` yet.** `apps/relay` is a fourth workspace module (already in local `go.work`) with no Dockerfile and no entry in `publish-images.yml`'s image matrix. When it's added, follow the same layering strategy below and add `COPY apps/relay/go.mod ./apps/relay/` to the `api`/`utils` Dockerfiles' sibling-module step (see "go.work Sibling Module Requirement"). Tracked in `docs/project-notes/issues.md` #130.

Related files at workspace root:
- `.dockerignore` — controls what gets sent to the Docker daemon
- `docker-compose.yaml` — builds and runs `api` + `utils`. Postgres is managed externally (not in compose); see ADR-016 amendment
- `.env` (gitignored) / `.env.example` — DB connection values consumed by both services via `env_file`

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

## go.work Sibling Module Requirement

`go.work` references all three modules. When building a single service (e.g. `api`), the other modules' source code isn't needed — but `go.work` still expects their `go.mod` to exist so `go mod download` can validate all workspace members. Each Dockerfile copies only the `go.mod` from sibling modules (no `go.sum`, no source):

```dockerfile
# utils source not needed for the api build, but go.work requires go.mod
COPY utils/go.mod ./utils/
```

Only the target service's source and `internal/shared/` are copied after the dependency cache layer.

## Dummy `config.json` for utils

`utils/main.go` uses `//go:embed configs/config.json` — the file must exist at build time or the embed fails. Since `config.json` is gitignored, the utils Dockerfile creates a dummy after copying source:

```dockerfile
RUN echo '{}' > ./utils/configs/config.json
```

At runtime, `NewDbConfig` parses `{}` successfully into a zero-value `DbConfig`, detects the zero value, and falls back to env vars (see BUG-023 — the zero-value check was added because `{}` does not produce a parse error). See issue #105 for a potential future cleanup of this pattern.

## Dockerfile Layering Strategy

1. Copy `go.work` + all `go.mod`/`go.sum` files
2. Copy sibling module `go.mod` for workspace validation
3. `go mod download` — cached unless dependencies change
4. Copy only the target service's source + `internal/shared/`
5. Create build-time workarounds (e.g. dummy `config.json` for utils)
6. Build the binary
7. Runtime stage copies only the binary
