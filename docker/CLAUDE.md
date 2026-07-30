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

**No `docker/relay/Dockerfile` yet.** `apps/relay` is a fourth workspace module with no Dockerfile and no entry in `publish-images.yml`'s image matrix — so it is built and tested in CI, but no image is ever produced or deployed for it. When one is added, follow the same layering strategy below. The sibling-module step it needs (`COPY apps/relay/go.mod ./apps/relay/`) is **already present** in both the `api` and `utils` Dockerfiles. Tracked in `docs/project-notes/issues.md` #130.

Related files at workspace root:
- `.dockerignore` — controls what gets sent to the Docker daemon
- `docker-compose.yaml` — builds and runs `api` + `utils`. Postgres is managed externally (not in compose); see ADR-016 amendment
- `.env` (gitignored) / `.env.example` — DB connection values consumed by both services via `env_file`

## Build Context

The build context is **always the workspace root**, not this directory. The Go workspace (`go.work`) and all four modules (`apps/api/`, `apps/utils/`, `apps/relay/`, `internal/shared/`) must be accessible during the build.

> **`docker/` did not move when the modules did** (ADR-022). Dockerfiles still live at `docker/<service>/Dockerfile` — only the `COPY` paths *inside* them gained the `apps/` prefix. `docker-compose.yaml` and `publish-images.yml`'s `-f docker/${service}/Dockerfile` were therefore unaffected by the move.

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
- **`utils`:** runs migrations and exits (not a long-running service). Since ADR-023 it embeds nothing and needs no build-time config file — it reads the seven `DB_*` keys from the environment at runtime like every other app.

## go.work Sibling Module Requirement

`go.work` references all four modules. When building a single service (e.g. `api`), the other modules' source code isn't needed — but `go.work` still expects their `go.mod` to exist so `go mod download` can validate all workspace members. Each Dockerfile copies only the `go.mod` from sibling modules (no `go.sum`, no source):

```dockerfile
# utils source not needed for the api build, but go.work requires go.mod
COPY apps/utils/go.mod ./apps/utils/
COPY apps/relay/go.mod ./apps/relay/
```

**Adding a module means editing every other service's Dockerfile**, not just adding a new one — miss a sibling `COPY` and `go mod download` fails on the *other* services' builds.

Only the target service's source and `internal/shared/` are copied after the dependency cache layer.

## `.dockerignore` patterns are path-anchored

A pattern without `**` is matched against the **full path relative to the context root**, not against each path segment. This is *not* gitignore's behavior, and the difference bites:

| Pattern | Excludes | Does **not** exclude |
|---------|----------|----------------------|
| `.env` | `<root>/.env` | `apps/api/configs/.env` |
| `dev.env` | `<root>/dev.env` | `apps/utils/configs/dev.env` |
| `**/dev.env` | every `dev.env` at any depth | — |

**Known issue:** every app now keeps a gitignored `configs/dev.env` with real credentials, and the rules as written do not exclude them. `COPY apps/<svc>/ ./apps/<svc>/` therefore pulls them into the **builder** stage. The final images are unaffected — multi-stage copies only the compiled binary out — so this is not a shipped-image leak, but the credentials are present in the build context and the builder layer, which matters if builder layers are ever exported to a registry cache or someone builds `--target builder`. Verify with a throwaway `COPY apps/utils/configs/ /x/` + `RUN ls /x/` rather than reading the patterns and assuming.

## Dockerfile Layering Strategy

1. Copy `go.work` + all `go.mod`/`go.sum` files
2. Copy sibling module `go.mod` for workspace validation
3. `go mod download` — cached unless dependencies change
4. Copy only the target service's source + `internal/shared/`
5. Build the binary
6. Runtime stage copies only the binary

Step 5 used to be preceded by a build-time workaround step (manufacturing a dummy `config.json` so `utils`' `//go:embed` would compile). ADR-023 removed the embed, and with it the only such workaround — no Dockerfile fabricates files anymore.
