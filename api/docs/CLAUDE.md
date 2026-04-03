# CLAUDE.md — api/docs

## Overview

This directory contains generated API documentation and Postman integration assets.

## Files & Directories

| Path | Purpose |
|------|---------|
| `swagger.json` | OpenAPI spec — **generated, do not edit by hand** |
| `swagger.yaml` | OpenAPI spec (YAML mirror) — **generated, do not edit by hand** |
| `docs.go` | Generated Go file that embeds the spec for `gin-swagger` |
| `postman/` | Postman-managed assets, synced to the git repo via Postman's Git integration |

## Regenerating Swagger Docs

After any change to handler annotations, regenerate from the `api/` directory:

```bash
(cd api && go generate ./...)
```

This runs `swag init -g main.go --output docs` and overwrites `swagger.json`, `swagger.yaml`, and `docs.go`.

## Postman

The `postman/` directory is managed by Postman and tied to this git repo. Its subdirectories map to Postman workspaces:

| Directory | Contents |
|-----------|----------|
| `collections/` | API request collections — auto-generated from `swagger.json` |
| `environments/` | Environment configs (e.g. base URL, non-secret vars) |
| `flows/` | Postman Flows definitions |
| `globals/` | Global variable definitions |
| `mocks/` | Mock server configs |
| `specs/` | Linked API spec snapshots |

### Secrets

Credentials and sensitive environment variables (API keys, tokens, passwords) are stored in the **Postman Vault** — not in this repo. A `.gitignore` in this directory ensures they are never committed.

Do not add secrets to any file under `postman/environments/` or `postman/globals/` — use the Postman Vault instead.
