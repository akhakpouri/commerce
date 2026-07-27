# Work Log

## Issue #105 — Retire the `utils` `config.json` embed (ADR-023)

**Date:** 2026-07-27
**Status:** Code complete; docs landed with this entry
**Branch:** `feature/issue-147` (done inline — see the note on #147 below)
**GitHub Issue:** #105
**Spec:** `docs/superpowers/specs/2026-07-27-utils-env-config-design.md` · **Plan:** `docs/superpowers/plans/2026-07-27-utils-env-config.md`

Dropped `//go:embed configs/config.json` from `utils` and moved all three apps onto one env-var config path in `internal/shared/configs`. Driven by the K8s Job work in #147: the embed forced a dummy `{}` file into both the Docker build and CI, and it was about to force more awkwardness into the Job.

**The decisive fact:** the embed was already dead code in every deployed environment. ECS injected `DB_*` and the `{}` dummy always parsed to a zero-value struct, so the env-var path is what actually ran. Removing it changed no production behavior.

- [x] `//go:embed` + `embed.FS` removed from `apps/utils/main.go`
- [x] `apps/utils/internal/managers/config_manager.go` deleted (with it, BUG-023's zero-value check)
- [x] `apps/utils/install.sh` deleted — ADR-006 retired; its "credentials baked in, run from anywhere" property is gone
- [x] `config.json` / `config.example` deleted; `dev.env` + `dev.env.example` added, matching `api` and `relay`
- [x] Dockerfile dummy-config line and the CI `Stage utils embed config` step removed
- [x] `GetEnvOrDefaultToInt` bug fixed — it converted the *key* instead of the value, so every one of relay's `ConsumerConfig` tuning vars (`Max`, `Timeout`, `WaitTime`, `Count`) was silently pinned to its default
- [x] `DatabaseConfig` wrapper dropped; all three apps now use gorm-kit's `db.DbConfig` directly
- [x] Docs + ADR-023
- [ ] **`.dockerignore` still lets `dev.env` into the build context** — the rule is `dev.env`, but Docker matches the full relative path, so it needs `**/dev.env`. Verified empirically: a probe build showed `dev.env` present under `apps/utils/configs/`. Final images are unaffected (multi-stage copies only the binary), but the credentials reach the builder layer.
- [ ] **`internal/shared/configs.NewConfig()` still inlines its own copy** of the seven-key block instead of calling `NewDbConfig()` — `api` and `relay` were deduped, this one was missed. `utils` and the other two therefore read config through two identical-by-coincidence code paths.
- [ ] No tests. `apps/utils` still has none, so the new required-key behavior is unverified. Task 2 of the plan has a table-driven `t.Setenv` suite ready to lift.

**Caught during review, before push:** CI would have failed outright — the `Stage utils embed config` step still ran `cp apps/utils/configs/config.example …` after `config.example` was deleted (`cp` exit 1 → `build` job red → PR blocked by the required check).

---

## Issue #147 — Orchestration: `api` + Postgres on minikube (ADR-021)

**Date:** 2026-07-24 (opened)
**Status:** In progress — Phase 3 of a personal K8s/CKAD learning plan. `api` + Postgres both up and talking to each other in-cluster; remaining work is `utils`, the rolling-update exercise, and Helm.
**Branch:** `feature/issue-147`
**GitHub Issue:** #147

Deploy `api` + in-cluster Postgres to a local minikube cluster as hands-on CKAD prep, using this repo as the anchor project instead of throwaway exercises. Target environment is minikube, not EKS — see ADR-021 in `decisions.md` for the full rationale (plain manifests before Helm, StatefulSet decided upfront, `utils`-as-Job pulled forward as a deliberate addition, namespace split, imperative DB provisioning). User is implementing and running all `kubectl`/manifest work themselves; my role is teaching/guide only — same posture as #114/#115/#143, extended here to explicitly exclude writing any manifests or code. Manifests live at `deployment/k8s/commerce/api/` and `deployment/k8s/postgres/` (see ADR-021's layout note — actual layout nests under `commerce/`, not flat per-component as first drafted).

- [x] `commerce` namespace created, set as default context namespace
- [x] `database` namespace created for the shared Postgres instance (ADR-021 namespace-split amendment — named `database`, not `postgres` as first discussed)
- [x] Dockerfile for the Go API (multi-stage build) — reused `docker/api/Dockerfile` (#97) as-is, confirmed working end-to-end (built via `eval $(minikube docker-env)`, tagged `commerce/api:local`)
- [x] Postgres in-cluster as a **StatefulSet** (not Deployment) + PVC — `postgres-app` StatefulSet, `volumeClaimTemplates` (not a manually pre-created PV), headless `postgres-service`, custom `local-storage` StorageClass (`k8s.io/minikube-hostpath` provisioner, `reclaimPolicy: Retain`) so PVs are born `Retain` instead of the default `standard` class's `Delete`
- [x] `commerce` database + a scoped app role created inside Postgres via `kubectl exec` + `psql` as superuser (imperative provisioning per ADR-021 — no init script/Job automation)
- [ ] `utils` (migration binary) as a **Job** against that Postgres, then converted to a **CronJob** as a learning variant — not started
- [x] DB credentials as a Secret (`commerce-secret`), non-secret config as a ConfigMap (`commerce-config`) — both in `commerce`; scoped DB creds point cross-namespace at `postgres-app-0.postgres-service.database.svc.cluster.local`
- [x] ClusterIP Service for API ↔ Postgres communication — headless `postgres-service` in `database` (see BUG-031: initial selector mismatch gave it zero Endpoints, fixed)
- [x] Liveness probe (process alive) — `GET /health/status/live`, passing
- [ ] Readiness probe with a real DB connection check — currently reuses the same liveness endpoint (process-alive only, not a real DB check); explicitly still open per the issue's original ask
- [ ] Resource requests/limits set deliberately, not copy-pasted defaults — values present in `deployment.yaml` (64Mi/250m requests, 128Mi/500m limits) but not yet reasoned through/right-sized together
- [ ] Rolling update on the API triggered while writes are in-flight — reason through what happens to an in-progress aggregate write during pod termination — not started
- [ ] (Optional) Expose the API via NodePort or a basic Ingress — explored `minikube service`/`minikube tunnel`; both require a standing local process on the `docker` driver (macOS), same constraint as `kubectl port-forward`. Decided: `kubectl port-forward` on demand is sufficient for this local orchestration exercise (the real dev workflow already uses a separate docker-compose Postgres) — not pursuing NodePort/Ingress further unless that changes.
- [ ] Once the above works end-to-end: convert to a Helm chart via `helm create`, parameterizing incrementally (image tag → replica count → resource limits)

**Bugs hit and fixed along the way:** BUG-031 (Service selector mismatch → zero Endpoints → DNS NXDOMAIN), BUG-032 (`:latest` + default `imagePullPolicy: Always` → `ImagePullBackOff` despite image present locally), BUG-033 (`server_address: localhost:8080` → unreachable from outside the container), BUG-034 (liveness/readiness probes on the wrong port/path) — see `bugs.md`.

**Explicitly out of scope this phase** (later phases per ADR-021): Auth0 integration, outbox `relay`/SNS-SQS wiring, EKS/cloud deployment, NetworkPolicies, resource quotas.

**Layout refactor landed on this branch (2026-07-27, ADR-022) — unrelated to the K8s work.** Before starting the `utils` Job, `api/` and `utils/` were moved under `apps/` to join `apps/relay`, closing ADR-018's open path-convention sub-decision. Done first so the `utils` manifests and Dockerfile edits were written against final paths rather than being touched twice. It accounts for the large rename diff on `feature/issue-147`; the K8s manifests themselves were unaffected (they reference image names, not source paths). Also closed the #130 CI gap as a ride-along — `apps/relay` is now in `go work init` and the build/test steps. Two things were caught pre-push: a dead path-anchored `.gitignore` rule that would have committed `apps/utils/configs/config.json`, and the relay CI additions initially being missed. A batch of unrelated dependency upgrades (including `lestrrat-go/jwx/v3` 3.1.1→3.2.0) rode along in the same commit by choice.

---

## Issue #143 — Event-sourced Order aggregate — Postgres event store (ADR-020)

**Date:** 2026-07-13 (opened)
**Status:** Design approved, not started
**Branch:** none yet (user will branch when they start coding)
**GitHub Issue:** #143

Isolated event-sourced `Order` aggregate + generic Postgres `EventStore` in `internal/shared/eventsourcing/` — raise/mutate/rehydrate, optimistic concurrency via `UNIQUE(stream_id, version)`, one upcaster template (`OrderPlaced` `TotalCents`→`TotalAmount`). Deliberately **not** wired into the live `OrderService`/handlers/routes/outbox — see ADR-020 in `decisions.md` for why (blocked on `GetByUserId` needing a projection layer that's out of scope). Full design: `docs/superpowers/specs/2026-07-13-event-sourced-order-aggregate-design.md`.

User is implementing this themselves; my role is plan + review, same posture as #114/#115.

- [ ] `EventStore` (`Append`/`Load`/`LoadSince`, `ErrConcurrencyConflict`)
- [ ] Events table registered in `internal/shared/database/main.go`'s `AutoMigrate` list
- [ ] `Order` aggregate: `OrderPlaced` + `OrderStatusChanged`
- [ ] Upcaster template
- [ ] Tests: in-memory fake (rehydration, upcasting, invalid transitions) + Postgres-integration (append, concurrency-conflict + reload-retry)

---

## Issue #130 — Transactional outbox + `relay` app (ADR-018 first slice)

**Date:** 2026-07-01 (opened) → 2026-07-10 (in progress)
**Status:** In progress
**Branch:** `feature/issue-130`

First implementation slice of ADR-018: the `commerce.outbox` table + the standalone `relay` app that drains it. Relay concurrency is **Model B** (autonomous workers, per-worker tx) — see the ADR-018 amendment (2026-07-01) in decisions.md and the "Relay concurrency" block in facts.md.

- [x] `commerce.outbox` GORM model + migration (via `utils`) — commits `2b53d6b`, `965a90d`
- [x] `internal/shared/repositories/outbox` — repository interface + GORM impl
- [x] `apps/relay` service layer — `OutboxService.ProcessBatch` claims via `GetNextBatch` (`FOR UPDATE SKIP LOCKED`) inside one `manager.Execute` transaction, then bulk `MarkPublished`. **Claim + mark only right now — nothing publishes anywhere yet** (see the SNS-vs-SQS open question below).
- [x] `apps/relay` polling daemon (`worker/daemon.go` + `main.go`) — single-process ticker loop; `signal.NotifyContext`-driven graceful shutdown; `daemon.Run`'s error is checked and `context.Canceled` is distinguished from a real failure via `errors.Is`, so a clean SIGINT/SIGTERM doesn't log as an error.
- [x] Verified safe at N replicas: ran two `go run .` processes against the same DB — `FOR UPDATE SKIP LOCKED` gives each a disjoint row set, no double-claim, matches the "safe at 1 or N replicas, no leader election" design in facts.md.
- [x] `apps/relay` added to local `go.work`
- [ ] ⚠️ Still not wired into CI: `.github/workflows/go.yml`'s `go work init ./api ./internal/shared ./utils` step doesn't include `./apps/relay` — CI never builds/tests it. Same gap in `publish-images.yml`. No relay `Dockerfile` yet.
- [ ] Publish plumbing — **design in flux, see below.**

### `internal/shared/aws` + `internal/shared/configs` (added 2026-07-10)

User-designed (not scaffolded) SQS producer/consumer toolkit — see the `aws`/`configs` sections in `internal/shared/CLAUDE.md` for the shape. Reviewed and fixed this session: 6 bugs found, see `bugs.md` BUG-024 through BUG-029 (WaitGroup miscounting, nil-slice panic risk on malformed messages, AWS config defaults that silently pointed non-local environments at LocalStack with placeholder credentials, SQS `DataType` casing, missing `Timeout`/`Url` validation). **Not yet wired into `apps/relay`** — `OutboxService.ProcessBatch` doesn't call any of it yet.

**Resolved 2026-07-21 — direct SQS vs. SNS fan-out.** Decided: **SNS**, matching the original design and the Terraform already drafted below (ADR-018 amendment 2026-07-21). This package's direct-SQS producer/consumer (no SNS client) was wired into `apps/relay`'s `OutboxService.ProcessBatch` in the interim (`apps/relay/internal/publisher.SqsPublisher`, resolving a queue literally named `commerce-queue`) — that needs replacing with an SNS publisher (topic ARN, `sns:Publish`) before this goes live. **Not yet done** — tracked as follow-up under this issue. Same session also banked a related pattern: cross-service ordering dependencies (e.g. payment must clear before shipping/notification act) are modeled as event chains, not a shared subscription — see the ADR-018 amendment and `facts.md`'s "Cross-service sequencing" section.

**`VisibilityExtender` — known gaps, discussed but not yet fixed:**
1. `StartVisibilityHeartbeat` isn't connected to `Consumer.process()` — no per-message start/stop lifecycle yet. Needs its own `context.WithCancel` scoped to one message's processing window, canceled the instant the handler returns (success or failure).
2. No retry on a single failed `ExtendVisibility` call — one transient AWS error silently ends renewal for the rest of that message, reopening the double-delivery race the mechanism exists to close.
3. No cap on total extension time — a hung (not crashed) handler would renew forever, making that message permanently unrecoverable. AWS's own hard ceiling is 12h from original receipt, but the app should cap well before that.

**Terraform (cross-repo, `~/code/infrastructure-code/matrix`, `aws/commerce/sns-sqs.tf`):** written 2026-07-06 — SNS topic `commerce-domain-events`, SQS queue `commerce-notifications-queue` + DLQ, queue policy (SNS → SQS delivery), subscription with `event_type` filter. `terraform plan` verified clean (5 to add, 0 changed/destroyed). User has since created the SNS topic and an SQS queue directly — exact names/shape not yet reconciled against this draft. Only the notifications queue/consumer (`notifier`) is in current scope per the "First slice" plan; billing/shipping queues are deferred until those services have actual side effects to run (avoids provisioning consumers with no code behind them).

**Note:** without the CI workspace wiring above, a regression in `apps/relay` wouldn't be caught until someone runs it manually — nothing in CI touches this module yet.

## Issue #127 — Integrate gorm-kit (retire in-tree DB connection logic)

**Date:** 2026-06-25
**Status:** Done
**Branch:** `feature/ci-ecs-deploy-job` (work landed alongside)

Swapped the hand-rolled `Connect`/`Migrate`/`DbConfig` boilerplate in `internal/shared/database` for the standalone, Go-proxy-published module **`github.com/akhakpouri/gorm-kit` v1.0.0** (extracted earlier; repo at `~/code/go/gorm-kit`). Amends ADR-015 — see decisions.md.

- [x] `api/go.mod`, `utils/go.mod`, `internal/shared/go.mod` — `require github.com/akhakpouri/gorm-kit v1.0.0`; `go work sync`; per-module `go mod tidy`
- [x] `internal/shared/database` reduced to a thin shim — removed `setup.go` + `db_config.go`; `main.go` `Migrate(cfg)` now delegates to `pg.Connect` + gorm-kit `database.Migrate(db, <models>)`. **Model registration list lives in the shim** (next to the models), so `utils/main.go` was left untouched.
- [x] `api/configs/config.go` — `databaseConfig.Connect()` calls `pg.Connect(database.DbConfig{...})` from gorm-kit (aliased imports `pg` + `db`)
- [x] `utils/internal/managers/config_manager.go` — `DbConfig` references repointed to gorm-kit's `database.DbConfig`; env-var fallback logic stayed (gorm-kit owns connection, not config loading)
- [x] Deps: `gorm.io/driver/postgres` dropped to `// indirect` (now via `gorm-kit/pg`); `gorm.io/gorm` stays **direct** (11 packages import `*gorm.DB`). `utils` gained its first external dep → new `utils/go.sum`.

**Gotchas hit:**
- Importing the bare module root `github.com/akhakpouri/gorm-kit` fails with an undeclared-name error — the root is a doc-only `package gormkit`. Must import the leaf packages (`/pg`, `/database`).
- `api/go.mod` initially marked gorm-kit `// indirect` despite a direct import (tidy not run in `api`) — fixed by `go mod tidy`.

**Verified:** build + tests pass on all three modules; migrate-gate non-zero-exit-on-failure preserved (`log.Fatal` → `os.Exit(1)`).

## Issue #113 — Auth0 JWT validation middleware

**Date:** 2026-05-05 → 2026-05-13
**Status:** Done (PR #118 open)
**Branch:** `feature/issue-113`

First implementation slice of ADR-017. Wires Auth0 config into the API and lands the JWT validation middleware end-to-end. Unblocked by iac-matrix#6 (Auth0 tenant + API resource server landed via Terraform 2026-05-05).

- [x] `api/configs/dev.env.example` — `AUTH_DOMAIN`, `AUTH_AUDIENCE` keys added
- [x] `api/configs/dev.env` — local values populated (gitignored)
- [x] `api/internal/constants/constants.go` — `EnvKeys.AuthDomain`, `EnvKeys.AuthAudience` added to typed struct
- [x] `api/configs/config.go` — exported `AuthConfig` struct (`Domain`, `Audience`); wired into `Config` via `NewConfig()` + `GetEnvOrPanic`
- [x] `api/go.mod` — `github.com/auth0/go-jwt-middleware/v3` added (note: **v3**, not v2 as originally planned)
- [x] `api/internal/auth/claims.go` — stateless `Claim{Scope string}` type implementing v3's `validator.CustomClaims` interface; `HasScope` helper
- [x] `api/internal/auth/validator.go` — `NewValidator(domain, audience)` with `jwks.NewCachingProvider` (5-min TTL), RS256, issuer with trailing slash
- [x] `api/internal/auth/middleware.go` — `NewMiddleware` + `Gin()` adapter: bearer extraction, JSON-formatted 401 error handler, success-path sets `*Identity` on `c.Set(constants.ContextKeys.Identity, ...)`
- [x] `api/internal/auth/identity.go` + `scope.go` — `Identity` context value; typed `Scopes` constants matching iac-matrix
- [x] Unit tests — `middleware_test.go` (HS256 with stdlib HMAC signing — no JWKS server needed) covering missing/invalid/expired/wrong-issuer/valid + scope parsing; `claims_test.go` for `Validate` + `HasScope`. 17 cases total.
- [x] Swagger `@securityDefinitions.apikey BearerAuth` in `main.go`; `@Security BearerAuth` on `/api/auth/whoami` and all 5 order endpoints; regenerated `docs/`
- [x] `/api/auth/whoami` debug endpoint (`api/internal/handlers/auth/`, `api/internal/dto/auth/who_am_i.go`) — returns subject + scopes + expiry for E2E verification
- [x] **Path mismatch fixed**: order handler `@Router` annotations were singular (`/api/order/...`) but actual Gin prefix is plural (`/api/orders`) — all 6 paths corrected
- [x] E2E verified 2026-05-13 with a real Auth0 M2M token (Test Application — since deleted, see facts.md)

**Architectural choices made during implementation:**
- Code lives in `api/internal/auth/` (NOT `api/internal/middleware/auth/` as originally planned). Single package; no `middleware/` parent.
- `RequireScope` ended up at handler-level (inside `RegisterRoutes`) rather than centralized in `routes.go`. Keeps read/write classification next to the route — see `api/internal/CLAUDE.md` for the pattern.
- `Claim.Validate` whitespace checks kept as defense-in-depth. Auth0 won't issue malformed scope strings in practice, but the cost is two lines and they're tested.
- `AuthConfig` is exported while `serverConfig`/`databaseConfig` remain unexported. Pattern split is intentional and fine; revisit when a second `AuthConfig` consumer lands.

---

## Issue #114 — Scope-check guard + per-route classification

**Date:** 2026-04-27 (opened) → 2026-05-13 (partially landed in #113)
**Status:** Partially done — guard exists and is wired on `orders`; remaining domains still unauthenticated
**Branch:** —

Per-route guard that asserts the JWT has a required scope before the handler runs. Consumes the `*Identity` stashed on `*gin.Context` by #113.

- [x] `RequireScope("orders:write")` helper — landed in #113 as `auth.RequireScope` in `api/internal/auth/middleware.go`
- [x] Typed scope constants in `auth.Scopes` (avoids string literals at call sites)
- [x] `authedApi` group exists in `routes.go` for routes that require a valid JWT
- [x] `orders` handler — all 5 routes behind `authedApi` + per-route `RequireScope`
- [ ] **Remaining domains still unauthenticated** — wire `authedApi` + `RequireScope` for: `address`, `category`, `tax`, `payment`, `products`, `user`, `review`. Policy: every route gets a scope (`:read` on GET, `:write` on mutation). `address` uses `users:*` (no `address:*` scope exists). No anonymous endpoints — see policy decision 2026-05-18.
- [ ] Nested routes (`/users/:user_id/addresses`, `/users/:user_id/orders`, `/orders/:id/payments`, `/products/:id/reviews`) — apply `:read` scope of the leaf resource (or `users:read` for the `users/:user_id/*` nests).
- [ ] Reconcile `users:delete` scope with ADR-011 (only delete-class scope; likely gates soft-delete; consider rename to `users:deactivate` upstream in iac-matrix)
- [ ] Address pluralization inconsistency in scope names AND route prefixes (`category`/`payment` singular vs `orders`/`products` plural). One coordinated pass on routes + scopes + annotations is cheaper than fixing in pieces.
- [ ] Unit tests per ADR-014 (E2E coverage exists via `middleware_test.go` `TestRequireScope_*` — add handler-level tests as scope guards land per domain)

---

## Issue #115 — Map Auth0 `sub` claim → domain `users` row

**Date:** 2026-04-27 (opened) → 2026-05-18 (design pinned) → 2026-05-20 (implementation complete)
**Status:** Done — PR pending
**Branch:** `feature/issue-115`

First request from a new Auth0 `sub` creates a `users` row mirroring identity from the token. Auth0 owns identity (login, password reset, email verification); this repo owns the domain user (orders, addresses, reviews).

### Design decisions (pinned 2026-05-18)

1. **M2M skip rule.** If `sub` ends in `@clients`, do NOT lookup-or-create. M2M tokens represent service clients, not people. The resolver middleware should `c.Next()` without populating `Identity.UserId` in that case — handlers that genuinely need a user can check `id.UserId == nil`.
2. **First-time payload from claims.** On create, populate `AuthSub` + `Email` + name fields from token claims. The frontend SPA must request `openid profile email` scopes so the JWT carries `email`, `given_name`, `family_name` (or `name`). The `Claim` struct in `api/internal/auth/claims.go` will be extended to deserialize these.
3. **No nullable Email.** `User.Email` stays `string` with `gorm:"unique"`. Since (2) guarantees an email at create-time, there's no NULL collision risk. If a token ever arrives without an `email` claim for a non-M2M `sub`, the resolver should fail loudly (401 or 500), not silently create a half-populated row.
4. **Two-middleware pattern.** A separate `ResolveIdentity(repo)` middleware runs after `Gin()`. Wiring:
   ```go
   authedApi := api.Group("", ginAuth, auth.ResolveIdentity(c.UserRepository))
   ```
   This keeps `Gin()` stateless/DB-free (existing tests don't break) and isolates the lookup-or-create concern. Inlining into `Gin()` was rejected.
5. **Extend `Identity` shape.** Add `UserId *uint` to `Identity` (nil for M2M, populated for resolved users). One context key, one cast at handler sites.

### Implementation checklist

- [x] `internal/shared/models/user.go` — added `AuthSub string `gorm:"unique;size:250"`` (dropped `not null` to keep AutoMigrate safe against existing rows; can tighten under #116 after bcrypt-flow cutover)
- [x] `internal/shared/repositories/user/user_repository.go` — `GetByAuthSub(sub string) (*models.User, error)` added to interface + impl
- [x] `api/internal/auth/claims.go` — extended `Claim` with `Email`, `FirstName` (json:"given_name"), `LastName` (json:"family_name"). Profile-claim validation lives in resolver, not `Validate()` — `Validate` can't tell M2M from user tokens.
- [x] `api/internal/auth/identity.go` — `UserId *uint` added (nil for M2M / unresolved)
- [x] `api/internal/auth/resolver.go` — `ResolveIdentity(svc UserServiceI)`: reads identity from ctx, M2M skip on `@clients` suffix, pulls claims via `ctx.Request.Context()` (not `ctx` — gin doesn't fall through for non-string keys), refuses empty email for non-M2M, delegates to service.
- [x] `api/internal/services/user/user_service.go` — `ResolveByAuth(sub, email, firstName, lastName)` added: hit returns existing DTO; miss builds `*models.User` directly (NOT through `dto.ToModel` — would lose GORM's populated Id), saves via repo, returns DTO via `FromModel`.
- [x] `api/server/router/routes.go` — chained `auth.ResolveIdentity(c.UserService)` after `ginAuth` in `authedApi` group
- [x] AutoMigrate verified on dev DB — `auth_sub` column added cleanly
- [x] Tests: `resolver_test.go` (6 cases: no Identity, M2M skip, missing email, happy path, service error, no claims context). `user_service_test.go` (4 cases: hit, miss-then-create, save-error+re-SELECT-finds-existing, save-error+re-SELECT-also-fails-propagates). Service-test miss case uses `DoAndReturn` to simulate GORM populating Id on insert and assert the surface.
- [x] `dto.User.AuthSub` exists with `json:"-"` so it round-trips internally without leaking into API responses
- **Race-on-create handling (resolved, not deferred):** concurrent first-touch requests for the same brand-new `sub` collide on the unique constraint. The loser's `Save` errors, and `ResolveByAuth` re-SELECTs by sub to return the winner's row instead of 500ing. A non-race `Save` error (re-SELECT also misses) still propagates. The repo-layer `OnConflict{DoNothing: true}` half of the original plan was not implemented — the service-layer re-SELECT covers the observable behavior on its own.

### Open implementation questions

- **Race on first-touch create.** Two requests for the same new `sub` race: both miss, both try to insert, one fails on unique index. Either (a) loser swallows the error and re-SELECTs, or (b) use GORM's `OnConflict{DoNothing: true}` clause + re-SELECT. (b) is cleaner.
- **Name field shape.** Auth0 typically issues `given_name` + `family_name` for federated logins, just `name` for username/password. Parse both; prefer `given_name`/`family_name` if present, fall back to splitting `name` on the first space. (Or just store both raw fields and don't split.)
- **What happens to the old `Authenticate(email, password)` flow?** Stays for now. Removed under #116 once Auth0 cutover is complete.

---

## Issue #116 — Deprecate `User.Password` + bcrypt hooks

**Date:** 2026-04-27 (opened) / pending — post-cutover follow-up
**Status:** Blocked on Auth0 cutover (after #113–#115)
**Branch:** —

Once Auth0 owns authentication, `User.Password` and the bcrypt `BeforeCreate` / `BeforeUpdate` hooks are dead weight. Remove them and supersede ADR-005.

- [ ] Drop `Password` field from `User` model
- [ ] Drop `BeforeCreate` / `BeforeUpdate` bcrypt hooks
- [ ] Drop `CheckPassword` method
- [ ] Drop `dto.Authenticate` and any handler/service code referencing it
- [ ] Migration to drop the `password` column
- [ ] Author ADR-018 (or amend ADR-005) marking ADR-005 superseded

---

## Issue #108 — ADR: authorization strategy (user JWT + OAuth 2.0 client credentials)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); see #113–#116
**Branch:** —

Original prerequisite for #109/#110, which described a build-in-tree auth approach. Pivoted to managed IdP — see ADR-017 in `decisions.md`.

---

## Issue #109 — User authentication (JWT bearer tokens)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); replaced by #113/#115
**Branch:** —

Originally would have built `/auth/login` and `/auth/register` in-tree against the existing `User` model + bcrypt. Auth0 owns these flows now (Universal Login).

---

## Issue #110 — OAuth 2.0 client credentials (M2M authorization)

**Date:** 2026-04-22
**Status:** Closed — superseded by ADR-017 revision (2026-04-27); replaced by #113/#114
**Branch:** —

Originally would have built `/oauth/token` with an `ApiClient` model + secret hashing in-tree. Auth0 M2M Applications cover this; no in-tree token issuer.

---

## Issue #99 — docker-compose.yaml for local development

**Date:** 2026-04-20
**Status:** In progress
**Branch:** feature/issue-99

Added `docker-compose.yaml` at the workspace root to build and run the `api` and `utils` containers with one command. Postgres is managed outside compose (see ADR-016 amendment).

- [x] `docker-compose.yaml` — `api` + `utils` services using `docker/api/Dockerfile` and `docker/utils/Dockerfile`
- [x] `api` uses `depends_on: utils` with `condition: service_completed_successfully` — waits for migrations to succeed before starting
- [x] Both services load config from root `.env` via `env_file`
- [x] `.env.example` committed as reference; `.env` gitignored
- [x] `utils/internal/managers/config_manager.go` — zero-value `DbConfig` now triggers env var fallback (see BUG-023)
- [ ] Verify end-to-end: `docker compose up` against externally-reachable Postgres
- [ ] Scrub example credentials in `.env.example` before committing

---

## Issue #98 — Dockerfile for utils migration runner

**Date:** 2026-04-15
**Status:** Done
**Branch:** feature/issue-98

Added `docker/utils/Dockerfile` — multi-stage build for the migration runner binary.

- [x] Multi-stage build: `golang:1.26-alpine` builder → `alpine:latest` runtime
- [x] Builder compiles from workspace root with `go.work` resolving all modules
- [x] Sibling module `api/go.mod` copied for workspace validation (no source)
- [x] Dummy `config.json` (`{}`) created during build so `//go:embed` succeeds — at runtime, JSON parse fails and `NewDbConfig` falls back to env vars (see #105 for future cleanup) — **superseded 2026-07-27: the embed and the dummy file are both gone, see ADR-023**
- [x] Runs as non-root user (`appuser`)
- [x] Image builds cleanly with `docker build -f docker/utils/Dockerfile .`

---

## Issue #97 — Dockerfile for api service

**Date:** 2026-04-14 → 2026-04-15
**Status:** Done
**Branch:** feature/issue-97

Added `docker/api/Dockerfile`, `.dockerignore`, `docker/CLAUDE.md`, and ADR-016.

- [x] Multi-stage build: `golang:1.26-alpine` builder → `alpine:latest` runtime
- [x] Sibling module `utils/go.mod` copied for workspace validation (no source)
- [x] Runs as non-root user (`appuser`)
- [x] `EXPOSE 8080` declared
- [x] `.dockerignore` at workspace root
- [x] `docker/CLAUDE.md` documenting build context, conventions, and layering strategy
- [x] ADR-016 — centralized Docker structure with workspace-root build context

---

## Issue #95 — Support env var config without dev.env file

**Date:** 2026-04-13
**Status:** Done
**Branch:** feature/issue-95

Removed the hard dependency on `api/configs/dev.env` so the binary starts correctly in containerized environments where env vars are injected at runtime (ECS Fargate / AWS Secrets Manager).

- [x] `configs/config.go` — `godotenv.Load` now wrapped in `os.Stat` check; only loads `dev.env` if the file exists; falls back to env vars silently
- [x] `api/configs/dev.env` — removed from git tracking (`git rm --cached`); added to `.gitignore`
- [x] `api/configs/dev.env.example` — committed as reference template with `DB_PASSWORD` left blank
- [x] `docs/project-notes/facts.md` — updated env vars section to reflect gitignored status and container fallback behaviour
- [x] `docs/project-notes/decisions.md` — updated ADR-004 config description and structure diagram; removed stale "known limitation" note

---

## Lint cleanup — all modules

**Date:** 2026-04-09
**Status:** Done

Ran `golangci-lint run ./...` across all three modules after all handlers were complete. Fixed 6 issues:

- [x] `address_handler.go` — `ineffassign`: `var address = &dto.Address{}` → `var address *dto.Address`
- [x] `category_handler.go` — `ineffassign`: `var products = []*product_dto.Product{}` → `var products []*product_dto.Product`
- [x] `tax_handler.go` — `ineffassign`: `states := []dto.Tax{}` + reassign → `var states []dto.Tax = h.svc.GetAll() //nolint:staticcheck` (explicit type required to keep `dto` import alive for swaggo — see BUG-017 edge case)
- [x] `order_handler.go` — `S1021`: merged `var statuses` declaration + assignment → `statuses := h.svc.GetStatuses()`
- [x] `payment_handler.go` — `S1021`: same merge
- [x] `config_manager.go` — `unused`: removed leftover `var content embed.FS` and `"embed"` import (see BUG-021)

---

## Issue #81 — Order handler

**Date:** 2026-04-09
**Status:** Done
**Branch:** feature/issue-81

Completed `OrderHandler` and wired the nested `/users/:user_id/orders` route.

- [x] `dto/order/order_status.go` — new `OrderStatus` DTO for `UpdateStatus` request body
- [x] `order_service.go` — added `GetStatuses() []dto.OrderStatus` to `OrderServiceI` and implementation
- [x] `order_handler.go` — `GetById`, `GetByUser`, `Save`, `Delete`, `GetStatuses`, `UpdateStatus` implemented with Swagger annotations
- [x] `order_handler.go` — `Delete` supports `?hard=true` query param consistent with other handlers
- [x] `routes.go` — `orderHandler.RegisterRoutes(api.Group("/orders"))` wired
- [x] `routes.go` — nested route `GET /api/users/:user_id/orders` wired to `orderHandler.GetByUser`

---

## Issue #84 — Review handler

**Date:** 2026-04-08
**Status:** Done
**Branch:** feature/issue-84

Completed `ReviewHandler` and wired the nested `/products/:id/reviews` route.

- [x] `review_handler.go` — `GetById`, `GetAllByProduct`, `Save`, `Delete` implemented with Swagger annotations
- [x] `review_handler.go` — `Delete` supports `?hard=true` query param consistent with other handlers
- [x] `routes.go` — nested route `GET /api/products/:id/reviews` wired to `reviewHandler.GetAllByProduct`; uses `:id` to avoid wildcard conflict with existing `GET /api/products/:id`

---

## Issue #82 — Payment handler

**Date:** 2026-04-08
**Status:** Done
**Branch:** feature/issue-82

Completed `PaymentHandler` and wired the nested `/orders/:order_id/payments` route.

- [x] `payment_handler.go` — `GetById`, `GetByOrder`, `Save`, `Delete`, `GetStatuses`, `UpdateStatus` implemented with Swagger annotations
- [x] `payment_handler.go` — `UpdateStatus` binds `dto.PaymentStatus` from request body; delegates validation to service
- [x] `payment_handler.go` — `GetStatuses` returns 404 if list is empty (in-memory — should never happen in practice)
- [x] `routes.go` — nested route `GET /api/orders/:order_id/payments` wired to `paymentHandler.GetByOrder`
- [x] `README.md` — updated status section to reflect all handlers complete; removed stale "hello, world!" note

---

## Issue #78 — User handler + nested address route

**Date:** 2026-04-06 → 2026-04-07
**Status:** Done
**Branch:** feature/issue-78

Completed `UserHandler` and wired the nested `/users/:user_id/addresses` route via `AddressHandler`.

- [x] `dto/user/authenticate.go` — new `Authenticate` DTO (email + password) for auth endpoints; `omitempty` removed from `Password` (see BUG-020)
- [x] `user_handler.go` — `GetById`, `GetAll`, `Authenticate`, `GetByEmail`, `Delete`, `Save` implemented with Swagger annotations
- [x] `user_handler.go` — `GetByEmail` returns 204 (existence check, no body — intentional)
- [x] `address_handler.go` — `GetByUserId` added; reads `user_id` param, delegates to `svc.GetAllByUserId`
- [x] `routes.go` — nested route `GET /api/users/:user_id/addresses` wired to `addressHandler.GetByUserId`
- [x] `api/internal/CLAUDE.md` — documented two-type `dto/user/` package and when to use each
- [x] `docs/project-notes/bugs.md` — BUG-020: `omitempty` on required fields silently drops values

---

## Issue #80 — Swagger annotation fixes + address handler completion

**Date:** 2026-04-02 → 2026-04-03
**Status:** Done
**Branch:** feature/issue-80

Fixed incorrect Swagger `@Router` path parameter syntax (`/:id` → `/{id}`) across all handlers, converted `address_handler` hard-delete to an optional query param, and completed the `AddressHandler` with a `Save` endpoint.

- [x] `address_handler.go` — route changed from `/:id/*hard` to `/:id`; `hard` now read via `c.DefaultQuery("hard", "false")`; `@Param hard query bool false` annotation added
- [x] `address_handler.go` — `@Router` updated to `{id}` OpenAPI syntax
- [x] `address_handler.go` — `POST /api/address` (`Save`) implemented with `@Param address body` annotation
- [x] `product_handler.go` — `@Router` updated to `{id}` for `GetById` and `Delete`; `@Param product body` added to `Save`
- [x] `category_handler.go` — `@Router` updated to `{id}` for `Delete`, `GetAllProductsByCategory`, `GetAllByParentId`, `GetById`; `@Param category body` added to `Save`
- [x] `api/internal/CLAUDE.md` — Swagger annotation rules updated with `{id}` vs `/:id` distinction, query param pattern for optional booleans, and tab-indented `@Param body` format

---

## Issue #80 — Category handler

**Date:** 2026-04-01
**Status:** Done
**Branch:** feature/issue-80

Completed `CategoryHandler` in `api/internal/handlers/category/category_handler.go`. Added `helpers.ParseParamToUint` to eliminate repeated `strconv.ParseUint` boilerplate.

- [x] `GET /api/category/` — `GetAll`
- [x] `GET /api/category/:id` — `GetById`
- [x] `GET /api/category/:id/children` — `GetAllByParentId`
- [x] `GET /api/category/:id/products` — `GetAllProductsByCategory` (delegates to `ProductService`)
- [x] `POST /api/category/` — `Save`
- [x] `DELETE /api/category/:id` — `Delete`
- [x] `api/internal/helpers/helper.go` — `ParseParamToUint` shared helper
- [x] Swagger docs regenerated; all endpoints annotated
- [x] `.vscode` untracked from git; `.gitignore` updated

---

## Issue #79 — Product handler

**Date:** 2026-04-01
**Status:** Done
**Branch:** feature/issue-79

Implemented the `ProductHandler` in `api/internal/handlers/product/product_handler.go` with full Swagger annotations. Wired DB connection via `container.Container` pattern (ADR-015). Also added `CategoryHandler` for the `GET /api/category/:id/products` endpoint (`GetAllByCategory` moved here per REST convention).

- [x] `GET /api/products/` — `GetAll`
- [x] `GET /api/products/:id` — `GetById`
- [x] `POST /api/products/` — `Save`
- [x] `DELETE /api/products/:id` — `Delete`
- [x] `GET /api/category/:id/products` — `GetAllByCategory` (CategoryHandler)
- [x] Container pattern implemented (`api/container/container.go`) — all services wired via `NewContainer(*gorm.DB)`
- [x] `api/internal/dto/err/error.go` — shared error response DTO
- [x] Swagger docs regenerated

**Note:** `GetAllByCategory` was intentionally placed in `CategoryHandler` (not `ProductHandler`) — nested resource convention (`/category/:id/products`).

---

## Issue #44 — HTTP handlers (Gin)

**Date:** 2026-03-26
**Last updated:** 2026-03-26
**Status:** In progress
**Branch:** feature/issue-73

Implement HTTP handlers using Gin (ADR-004). Service layer and DTOs are complete — handlers are the next layer.

- [x] Add `github.com/gin-gonic/gin` to `api/go.mod`
- [x] Wire router in `api/main.go`
- [x] `api/server/server.go` — `Server` struct with graceful shutdown (`Run()`)
- [x] `api/server/router/routes.go` — `RegisterRoutes()` composition root for HTTP layer
- [x] `api/configs/config.go` — env-based config, `GetEnvOrPanic`, CORS middleware
- [x] `api/internal/constants/constants.go` — typed env key + header name constants
- [x] `api/internal/handlers/tax/tax_handler.go` — first handler group (`GET /api/v1/taxes/`)
- [ ] Wire DB connection (`databaseConfig` exists but unused — needed before repo-backed handlers)
- [ ] Implement remaining handler groups (one per domain)

---

## Issue #69–#73 — Service layer unit tests (ADR-014)

**Date:** 2026-03-12
**Last updated:** 2026-03-26
**Status:** Done
**Branch:** feature/issue-73

- [x] #69 — Add test dependencies to `api/go.mod` (testify, bcrypt)
- [x] #70 — Unit tests for `TaxService`
- [x] #71 — Unit tests for `OrderService`
- [x] #72 — Unit tests for `UserService`
- [x] #73 — Unit tests for `PaymentService`

#69 must be completed before any test file work begins. See ADR-014 in `decisions.md` for mock strategy and full test case matrix.

**OrderService test cases (feature/issue-73):**
- `TestGetbyId` — happy path
- `TestGetbyIdError` — repo error propagation
- `TestGetAllByUser` — happy path, asserts count
- `TestGetAllByUserError` — repo error propagation
- `TestDelete` — soft delete
- `TestDeleteHard` — hard delete
- `TestDeleteError` — repo error propagation
- `TestSave` — verifies SubTotal/Tax/Total computed correctly before persist (DoAndReturn on model)
- `TestSaveInvalidState` — invalid BillingState → tax service errors → repo never called
- `TestUpdateStatus` — valid status → repo called
- `TestUpdateStatusInvalid` — invalid status → error, repo never called
- `TestUpdateStatusRepoError` — repo error propagation

**Key testing notes:**
- `Save` takes `dto.Order` by value — assert computed amounts inside `DoAndReturn` on the model, not on the caller's variable
- Use `assert.InDelta` for tax/total (floating point); `assert.Equal` is safe for subtotal (integer arithmetic)
- MD tax rate is `0.06` — subtotal 40.00 → tax 2.40 → total 42.40

---

## Issue #66 — Compute SubTotal, Tax, and Total in OrderService.Save

**Date:** 2026-03-11
**Status:** Done

- [x] Add `SubTotalAmount`, `TaxAmount`, `BillingState` to `Order` DTO; updated `ToModel`/`FromModel`
- [x] Inject `TaxServiceI` into `OrderService` constructor
- [x] Compute `SubTotalAmount`, call `TaxService.Calculate`, set `TotalAmount` in `Save`
- [x] `calculateTax` returns `(float64, error)` — no pointer, error propagated in `Save`
- [x] `GetById` in order repo preloads `BillingAddress` so `BillingState` is populated

---

## Issue #65 — Implement TaxService

**Date:** 2026-03-11
**Status:** Done

- [x] Create `api/internal/services/tax/tax_service.go`
- [x] `TaxServiceI` interface with `Calculate(amount float64, state string) (*float64, error)` and `GetStates() []string`
- [x] In-memory `map[string]dto.Tax` implementation; error on unknown state
- [x] `float64` throughout for precision
- [x] `GetStates` returns sorted keys

---

## Issue — Repository + Service layer implementation (ADR-008, ADR-009)

**Date:** 2026-02-27
**Last updated:** 2026-03-12
**Status:** Done
**Branch:** `feature/issue-22`

Implementing the repository layer (ADR-009) and service layer (ADR-008). See both ADRs in `decisions.md` for full interface signatures and implementation notes.

**Repository layer** (`internal/shared/repositories/`) — owns GORM queries, returns models:
- [x] `repositories/user/user_repository.go`
- [x] `repositories/address/address_repository.go`
- [x] `repositories/product/product_repository.go`
- [x] `repositories/category/category_repository.go`
- [x] `repositories/review/review_repository.go`
- [x] `repositories/order/order_repository.go`
- [x] `repositories/payment/payment_repository.go`

**Note — `Save` method primary key retention:**
GORM mutates the pointer passed to `Save` in place — the generated primary key is written back onto the struct automatically. No signature change needed. Callers just need to hold onto the pointer they pass in and read the ID from it after `Save` returns. No action required — awareness only.

**Service layer** (`api/internal/services/`) — owns business logic, returns DTOs:
- [x] `services/address/address_service.go`
- [x] `services/category/category_service.go`
- [x] `services/user/user_service.go`
- [x] `services/product/product_service.go`
- [x] `services/review/review_service.go`
- [x] `services/order-item/order_item_service.go`
- [x] `services/order/order_service.go`
- [x] `services/payment/payment_service.go`

**Repo additions required before services can be completed:**
- `user_repository.go` — add `GetByEmail(email string) (*models.User, error)` (needed by `UserService.Authenticate`)
- `order_repository.go` — add `GetByUserId(userId uint) ([]*models.Order, error)` (needed by `OrderService.GetByUserId`)

**Service design notes (feature/issue-26):**

`UserService` — interface: `GetById`, `GetAll`, `Save`, `Delete(id, hard)`, `Authenticate(email, password)`
- `Authenticate`: `repo.GetByEmail` → `model.CheckPassword(password)` → return `errors.New("invalid credentials")` if false

`ProductService` — interface: `GetById`, `GetAll`, `GetByCategory(categoryId)`, `Save`, `Delete(id, hard)`
- `GetByCategory` lives here (not CategoryService) — returns products; category is just a filter
- `GetByOrder` was considered and rejected — `OrderItem` DTO already carries the product info needed at order time; no need to re-fetch

`ReviewService` — interface: `GetById`, `GetByProductId`, `Save`, `Delete(id, hard)`
- `GetByProductId` returns `[]*dto.Review`

`OrderService` — interface: `GetById`, `GetByUserId`, `Save`, `Delete(id, hard)`, `UpdateStatus(id, status)`
- Injects both `OrderRepositoryI` and `OrderItemRepositoryI` (per CLAUDE.md)
- `UpdateStatus`: validate status string against `models.OrderStatus` consts before calling repo
- Valid statuses: `pending`, `shipped`, `delivered`, `cancelled`

`PaymentService` — interface: `GetById`, `GetByOrderId`, `Save`, `Delete(id, hard)`, `UpdateStatus(id, status)`
- `GetByOrderId` maps to `repo.GetByOrder`
- `UpdateStatus`: validate against `models.PaymentStatus` consts before calling repo
- Valid statuses: `pending`, `completed`, `authorized`, `captured`, `failed`, `refunded`, `partially_refunded`

**Consistency rules (follow address/category pattern):**
- Return `[]*dto.X` for slices
- Log errors with `slog.Error(...)` before returning
- Constructor returns the interface type
- Import alias: `userdto "commerce/api/internal/dto/user"`, `userrepo "commerce/internal/shared/repositories/user"` etc.

---

## Issue #38 — Payment model implementation

**Date:** 2026-02-26
**Status:** Done
**Branch:** `feature/issue-9`
**GitHub Issue:** #9

Designing and implementing the `Payment` entity as per ADR-007. Model lives in `internal/shared/models/payment.go` and must be registered in `internal/shared/database/setup.go`.

**Scope:**
- [x] `Payment` model with all fields from ADR-007
- [x] Register model for GORM AutoMigrate
- [x] Update `Order` model if needed (e.g., `Payments []Payment` association)

See ADR-007 in `decisions.md` for full field list and rationale.

---

## Issue #37 — ADR-003 embed fix

**Status:** Done
**Branch:** `feature/issue-37`

Resolved three bugs related to the `//go:embed` config setup (see BUG-002, BUG-003):

1. `config_manager.go` had `var content embed.FS` with no `//go:embed` directive — FS was always empty.
2. Embed responsibility was refactored: `NewDbConfig` now accepts `[]byte`; file reading and embedding moved to `utils/main.go`.
3. In `main.go`, the `//go:embed` directive was attached to `var _ embed.FS` (blank identifier) instead of `var content embed.FS` — fixed by moving the directive to the correct variable.
4. Fixed fallback logic: env var path now returns `nil` error so the caller can proceed.
5. Restored `utils/configs/config.json` as the canonical config location; updated `.gitignore` to match.

---

## Issue #34 — (merged)

**Branch:** `feature/issue-33`
**Merged commit:** `82a534f`
**Status:** Done

---

## Issue #33 — (merged)

**Status:** Done
**Notes:** Readme update included (`5c69c89`), config file removed (`109803b`).
