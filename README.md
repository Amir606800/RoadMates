# RoadMates

Mobile React app and Go API for shared rides between Azerbaijani districts.
Accounts, driver verification, district selection, and rides use PostgreSQL.
Redis accelerates public searches and shares rate limits between API instances.
Reviews remain a UI preview, the map is illustrative, and seat reservations and
payments are outside the implemented API. CI/CD is intentionally not set up.

## Architecture

```mermaid
flowchart LR
  Browser -->|HTTP :8081| Nginx[Frontend nginx + React]
  Nginx -->|/api, /healthz, /readyz| API[Go backend :8080]
  API --> PostgreSQL
  API --> Redis
  Migrate[Migration job] --> PostgreSQL
```

Only nginx publishes a host port. PostgreSQL, Redis, and the backend share a
private Docker network. PostgreSQL uses a named volume; Redis persistence is
disabled and losing its data is harmless. The backend starts after migrations
complete successfully and Redis is initially healthy; an outage after startup
uses cache and rate-limit fallbacks.

Backend layers are versioned HTTP handlers/DTOs → services/interfaces → pgx
repositories or Redis adapters. `backend/internal/api/v1` owns the v1 contract;
domain types do not carry JSON tags. The React auth provider owns the access
token in memory, and a shared API client handles session rotation.

## Run with Docker

Install Docker Desktop with Linux containers, Compose v2.10+, Node.js 22.12+,
and optionally GNU Make. Start Docker Desktop. In PowerShell at the repo root:

```powershell
Copy-Item .env.example .env
# Edit .env and replace all secret placeholders before starting.
make up
# Without Make: node scripts/compose.mjs up --build -d
```

Generate `JWT_SECRET` and `PII_ENCRYPTION_KEY` locally:

```powershell
node -e "console.log(require('node:crypto').randomBytes(48).toString('base64url'))"
node -e "console.log(require('node:crypto').randomBytes(32).toString('base64'))"
```

Use the first value for JWT signing and the second for PIN encryption. Keep the
PII key stable: existing applications need the key used when they were submitted.
Use URL-safe PostgreSQL/Redis passwords or percent-encode them in connection
URLs. Secrets are ignored by git and excluded from image build contexts.

Open <http://localhost:8081>. To choose another host port, change `WEB_PORT` in
the root `.env`; `ALLOWED_ORIGIN=http://localhost:${WEB_PORT}` follows it.
Internal backend/nginx ports remain 8080. Accessing through a phone or another
hostname requires that exact browser origin in `ALLOWED_ORIGIN`.

`COOKIE_SECURE=false` supports local HTTP. Set it to `true` when serving behind
HTTPS. The production images run as non-root users. nginx serves the SPA,
proxies the API on the same origin, applies security headers, caches hashed
assets immutably, and avoids caching `index.html` and `version.json`.

| Command | Behavior |
| --- | --- |
| `make up` | Build and start services and migrations. |
| `make down` | Stop containers; preserve PostgreSQL data. |
| `make logs` | Follow service logs. |
| `make rebuild` | Rebuild without layer cache and recreate services. |
| `make migrate-up` | Apply pending migrations. |
| `make migrate-down` | Revert one migration; dependent data can prevent rollback. |
| `make test` | Run frontend checks and Go race/integration/contract checks using disposable databases. |
| `make release VERSION=x.y.z` | Update VERSION, remind you to update CHANGELOG, build tagged images; no push. |

Shells without Make can use `node scripts/compose.mjs` followed by normal
Compose arguments, `node scripts/test.mjs`, and `node scripts/release.mjs x.y.z`.
For example, `node scripts/compose.mjs logs --tail=100 backend` shows API logs.

## Accounts and driver verification

Register at `/register`, then open Profile. Every new user automatically gets
the client role. Passwords require at least eight Unicode characters, at most
72 UTF-8 bytes, uppercase, lowercase, and punctuation/symbols, with no whitespace
or control characters. bcrypt cost is 12.

Set both `ADMIN_EMAIL` and `ADMIN_PASSWORD` before first startup to bootstrap a
new client + admin account. The password follows the same policy. Bootstrap
does not promote an existing account or reset its password. There is no public
admin-registration route. Administration appears in the admin's Profile.

From Profile → Become a driver, submit identity/contact and vehicle details.
The seven-character alphanumeric PIN is cleared from the form on submission
and encrypted using AES-256-GCM with a random nonce and user/application-bound
authenticated data. Only the admin detail response decrypts it; the screen
masks it until Show PIN. Applicant responses and admin lists omit it and its
ciphertext. Vehicle years run from 1980 to next year; passenger seats are 1–8.

Only one pending application per user is allowed. Admins can reject with a
reason or approve; approval grants driver while retaining client in the same
transaction. Concurrent reviews have one winner. Rejected users can correct
and resubmit. Pending status polls every ten seconds and on window focus;
approval triggers auth refresh to obtain the new role. Driver and admin route
guards use server-issued roles.

Access JWTs are HS256 with `sub`, `roles`, `iat`, `exp`, and `jti`. They are
stateless and stay only in React state/ref. They are never put in localStorage,
sessionStorage, IndexedDB, or readable cookies. Refresh tokens are random 32-byte
opaque values; PostgreSQL stores only SHA-256 hashes. The HttpOnly cookie is
SameSite=Lax, Path `/api/v1/auth`, and Secure according to configuration.

Each refresh revokes and replaces its token in one family. Reusing a revoked
token revokes the whole family. Refresh/logout require the exact allowed
`Origin` and `X-Requested-With: roadmates`. Startup refresh restores the session;
parallel 401s share one refresh and retry once, and proactive refresh runs
before expiry. Logout revokes the refresh family and clears local auth state;
already issued JWTs expire normally. A background job removes refresh tokens
expired more than seven days ago and stops during graceful shutdown.

## Districts and rides

All pickers load active districts from PostgreSQL. The migration seeds the 63
supplied entries, ordered by `sort_order` then name. ASCII `samaxi` finds Şamaxı;
search folds Azerbaijani letters and case. The nine permit districts remain
selectable with a hint. There are no hardcoded district selector lists.

Clients choose two different districts, date, and required seats. Drivers use
the same picker to publish a future departure, 1–8 seats, a price from 0 through
999999.99 AZN with at most two decimals, and an optional 300-character note.
Prices are parsed as integer cents and stored as PostgreSQL `numeric(8,2)`.
Search dates and displayed times use Azerbaijan time (`Asia/Baku`, UTC+04:00).
API timestamps include a timezone offset or `Z`.

A driver's active ride is `published` or `in_progress`, even if its departure
has passed. The publish service checks first; the partial unique index is the
concurrency guarantee. Both paths return `409 DRIVER_HAS_ACTIVE_RIDE`.
The driver screen checks `/rides/me/active` before showing a form. An existing
ride replaces the form with Cancel and Complete; either action allows another
publish. A failed lookup keeps publishing hidden, and a cross-tab conflict
refreshes the current ride. Only its owner driver can change it.

Public search returns published rides with available seats and future departure,
active drivers/districts, and real district names, ordered by departure then ID.
It accepts optional departure/destination/date filters. The frontend additionally
filters by the requested seat count. Details show saved ride data; no mock seat
purchase is performed.

## API contract and operational checks

The complete [OpenAPI 3.0.3 contract](backend/api/openapi/v1.yaml) describes all
implemented endpoints, request/response fields, roles, cookies, CSRF headers,
and error codes. Import it into an OpenAPI viewer or API client. Go tests validate
the document and check mounted paths, DTO fields, and error codes against code.
GET endpoints also accept HEAD. JSON bodies are limited to 16 KiB and reject
unknown fields, extra JSON documents, invalid media types, and invalid input.

| Endpoint | Access and behavior |
| --- | --- |
| `GET /healthz` | Liveness; independent of dependencies. |
| `GET /readyz` | PostgreSQL down → 503; Redis alone down → 200 degraded. |
| `GET /api/version` | App/build/Go versions and supported API/app versions. |
| `GET /api/v1/` | v1 index; `/api/v1` redirects here. |
| `POST /api/v1/auth/register`, `/auth/login` | Return access token + user; set refresh cookie. |
| `POST /api/v1/auth/refresh` | Cookie + CSRF; rotate and return current user/roles. |
| `POST /api/v1/auth/logout` | CSRF; revoke family, clear cookie, return 204. |
| `GET /api/v1/auth/me` | Client Bearer; current user and roles. |
| `POST /api/v1/driver-applications` | Client Bearer; submit pending application. |
| `GET /api/v1/driver-applications/me` | Client Bearer; latest application or 404. |
| `GET /api/v1/admin/driver-applications` | Admin Bearer; status, limit (1–100), offset filters. |
| `GET/PATCH /api/v1/admin/driver-applications/{id}` | Admin detail or approve/reject. |
| `GET /api/v1/districts?q=` | Public district search. |
| `GET/POST /api/v1/rides` | Public search or driver-only publish. |
| `GET /api/v1/rides/me/active` | Driver Bearer; active ride or 404. |
| `GET /api/v1/rides/{id}` | Public ride details. |
| `PATCH /api/v1/rides/{id}/cancel`, `/complete` | Owner driver Bearer; end active ride. |

All backend responses include `X-API-Version: 1` and `X-Request-ID`. A valid
incoming request ID is preserved; otherwise one is generated. Errors are JSON
`{code, message, request_id}`; JSON responses carry `Cache-Control: no-store`.
Structured slog request logs include method, path, status, duration, request ID,
user ID when known, app version, and client. Passwords, tokens, PINs, request
bodies, and connection URLs are never logged.

```powershell
curl.exe -i http://localhost:8081/healthz
curl.exe -i http://localhost:8081/readyz
curl.exe -i http://localhost:8081/api/version
curl.exe -i http://localhost:8081/api/v9/rides
curl.exe -i 'http://localhost:8081/api/v1/districts?q=samaxi'
curl.exe -i 'http://localhost:8081/api/v1/rides?from_district_id=9&to_district_id=48'
```

Unknown versions return 404 `API_VERSION_NOT_FOUND`. Missing `X-App-Version`
is allowed and logged; malformed semver returns 400 `APP_VERSION_INVALID`.
An app below `MIN_SUPPORTED_APP_VERSION` gets 426 `APP_VERSION_UNSUPPORTED`;
the frontend shows a reload/update screen. Health/readiness are exempt. To test
the screen, temporarily set the minimum to `0.2.0`, rebuild/start, then restore
`0.0.0` and rebuild/start again.

## Redis caching and rate limits

Services depend on a Cache interface (`Get`, `Set` with TTL, `Delete`, namespace
read/bump). Redis, in-memory, and no-op implementations are provided. Cache-aside
reads use singleflight to coalesce misses. Cancelling one reader does not cancel
the shared database read for other readers. Redis operations have short deadlines;
failures warn without logging raw errors/keys and fall back to PostgreSQL.

| Key pattern | Data and lifetime |
| --- | --- |
| `roadmates:v1:districts:<folded-query-hash>` | Active district DTO data, 24 hours. |
| `roadmates:v1:rides:version` | Search namespace counter; randomized initial value. |
| `roadmates:v1:rides:search:<version>:<filter-hash>` | Matching ride UUIDs only; 30–60 seconds with TTL jitter. |
| `roadmates:v1:ratelimit:<group>:<ip-hash>:<window>` | Fixed-window count with expiry. |

Ride hits hydrate names/notes from PostgreSQL and recheck status, availability,
departure, and active users/districts. No auth data, user profiles, driver
applications, PINs, or ride notes are stored in Redis. Publish/cancel/complete
increment the namespace after commit. Old search keys expire naturally; there
is no `KEYS`/`SCAN` invalidation. Random initialization avoids reusing old cache
entries if the namespace counter alone is evicted by `allkeys-lru`.

If invalidation fails, the process bypasses its search cache until it can retry
the bump. This pending state is in memory; after a process crash during an
outage, another instance can observe an older search membership until its
30–60 second TTL ends. Hydration still removes cancelled, departed, or full rides.
District changes made by a new migration become visible as cached entries expire
within 24 hours; restarting the cache-only Redis also discards those entries.

Rate limits use an atomic Lua `INCR` + `EXPIRE` operation, following the
[Redis counter pattern](https://redis.io/docs/latest/commands/incr/#pattern-rate-limiter).
Per-IP, per-minute limits are register **5**, login **10**, refresh **60**, and
driver application **10**. `429 RATE_LIMITED` includes `Retry-After` in seconds.
During a Redis outage a bounded local limiter applies; it shadows healthy
traffic to preserve the count already seen by that process. The fallback quota
is per process, so it is less strict across multiple instances during outages.
Docker trusts nginx's overwritten `X-Real-IP` because the backend is private.
Host mode defaults `TRUST_PROXY_HEADERS=false`.

To verify outage behavior on your local stack, stop Redis temporarily, repeat
district/search requests and readiness, then restore it:

```powershell
node scripts/compose.mjs stop redis
curl.exe -i http://localhost:8081/readyz
curl.exe -i 'http://localhost:8081/api/v1/districts?q=samaxi'
node scripts/compose.mjs start redis
```

Expect 200 degraded readiness and working district/ride requests. Stopping
PostgreSQL instead makes readiness return 503; liveness remains 200.

## Tests

```powershell
make test
# Without Make:
node scripts/test.mjs
```

The runner builds/tests the frontend, then uses `docker-compose.test.yml` with
the separate `roadmates-tests` project. Disposable PostgreSQL/Redis containers
have no published ports or application volumes. Go runs with the race detector;
the backend test image also runs `go vet`. The runner removes test containers
and volumes afterward. Its explicit test credentials are temporary fixtures.

Checks cover password policy, JWT algorithms/expiry, refresh rotation/reuse,
role boundaries, encryption and application review transactions, simultaneous
publishes, district folding, ride lifecycle/search, real Redis cache hits and
invalidation, cache outage/recovery, shared Redis rate limiting and local fallback,
singleflight cancellation, namespace eviction, OpenAPI validity/drift, unknown
API versions, outdated app versions, and dependency readiness. Frontend unit
tests cover session concurrency, ride payloads, timezone boundaries, validation,
and the retained standalone seat/booking preview helpers.

For quicker host checks, use `go test ./...` and `go vet ./...` in `backend`, and
`npm.cmd test` / `npm.cmd run build` in `frontend`. Database tests opt in with
`TEST_DATABASE_URL` and `TEST_REDIS_URL`. PostgreSQL tests create/drop unique
schemas; Redis tests use known test keys and never flush or enumerate the cache.
Use dedicated test databases, particularly when testing search invalidation.

Manual browser checks: register, submit a driver application, review in a
separate admin browser session, and confirm automatic driver-role refresh.
Publish a ride, search its districts/date as a client, reload the driver page,
complete/cancel, and republish. Keep two driver tabs open to check the 409 flow.
Check widths of 360, 390, and 430px and keyboard navigation in district dialogs.

## Host development

The development override publishes PostgreSQL/Redis on localhost only:

```powershell
node scripts/compose.mjs down
node scripts/compose.mjs -f docker-compose.yml -f docker-compose.dev.yml up -d postgres redis
node scripts/compose.mjs -f docker-compose.yml -f docker-compose.dev.yml run --rm migrate up
```

Go 1.23+ reads process environment variables, not dotenv files. Use
`backend/.env.example` as a host configuration reference. Export the same
credentials and stable PII key as your root `.env`, using localhost service
addresses and `ALLOWED_ORIGIN=http://localhost:5173`. For example:

```powershell
cd backend
$env:DATABASE_URL = 'postgres://roadmates:YOUR_URL_ENCODED_PASSWORD@localhost:5432/roadmates?sslmode=disable'
$env:REDIS_URL = 'redis://:YOUR_URL_ENCODED_REDIS_PASSWORD@localhost:6379/0'
$env:JWT_SECRET = 'YOUR_GENERATED_JWT_SECRET'
$env:PII_ENCRYPTION_KEY = 'YOUR_EXISTING_BASE64_KEY'
$env:ALLOWED_ORIGIN = 'http://localhost:5173'
go run ./cmd/server
```

In a second terminal:

```powershell
cd frontend
npm.cmd ci
npm.cmd run dev
```

Open <http://localhost:5173>. Vite proxies `/api`, `/healthz`, and `/readyz`.
Copy `frontend/.env.example` to `.env.local` for optional settings;
`VITE_PROXY_TARGET` changes the backend address. If port 8080 is already used,
export a different backend `PORT` and point Vite at it. Use `npm`/`curl` instead
of `npm.cmd`/`curl.exe` on macOS/Linux. Frontend `VITE_*` settings are public
build data and must never contain secrets.

## Versioning and migrations

`VERSION` is the app release source of truth. Compose/release helpers inject
it, the git commit (or `unknown` outside a checkout), and UTC build time. Backend
ldflags populate `internal/buildinfo`; Vite build arguments populate the Profile
footer and `/version.json`. Images receive both the version tag and local
`:latest`. `APP_VERSION` in `.env` is a fallback for direct Compose invocations.

To cut a release, review/update `CHANGELOG.md`, run tests, then run
`make release VERSION=x.y.z`. This updates VERSION and builds both images,
without publishing or pushing. The helper reminds you to update the changelog.
Review the OpenAPI contract whenever the API changes; run `make up` to start
the new local images. No GitHub Actions, GitLab CI, or other pipeline is present.

SQL migrations use golang-migrate and sequential `up`/`down` SQL files under
`backend/migrations`. **Never edit an applied migration.** Add the next numbered
migration instead. `000001` creates the schema/constraints and automatic client
role trigger; `000002` seeds districts. No later phase rewrites these migrations.
Check/apply changes with `make migrate-up`; `make migrate-down` reverts one
version and can fail when dependent rows exist.

API versioning is independent of app/database versions. To add v2, create
`backend/internal/api/v2` with its own DTOs/handlers and `NewRouterV2`, then
mount `/api/v2/` next to v1 in `internal/router/router.go`. Update the supported
versions in `buildinfo`, add `api/openapi/v2.yaml`, and keep v1 intact while
clients migrate. Shared domain/repository code can be reused through services.
v1 deprecation is off by default; enable `API_V1_DEPRECATED` with RFC3339
deprecation/sunset dates (sunset must follow deprecation) to emit headers.

## Environment reference

Root `.env` is runtime configuration for Docker; `.env.example` contains only
placeholders. Host backend settings are in `backend/.env.example`.

| Variable | Default/reference | Purpose |
| --- | --- | --- |
| `POSTGRES_USER`, `POSTGRES_DB` | `roadmates` | PostgreSQL container account/database. |
| `POSTGRES_PASSWORD` | Required | Database password. |
| `DATABASE_URL` | Required | pgx/migrate connection URL; Docker host is `postgres`. |
| `REDIS_PASSWORD` | Required in Compose | Redis container password. |
| `REDIS_URL` | Required | Password-authenticated URL; Docker host is `redis`. |
| `WEB_PORT` | `8081` | Only published browser-facing Docker port. |
| `PORT` | `8080` | Host backend listen port; Compose fixes it to internal 8080. |
| `APP_VERSION` | `0.1.0` fallback | Direct-Compose image tag; helpers use VERSION instead. |
| `MIN_SUPPORTED_APP_VERSION` | `0.0.0` | Minimum full semantic app version. |
| `LOG_LEVEL` | `info` | debug/info/warn/error structured logging. |
| `JWT_SECRET` | Required, ≥32 bytes | Non-placeholder HS256 signing secret. |
| `ACCESS_TOKEN_TTL` | `15m` | Duration from 1s through 24h. |
| `REFRESH_TOKEN_TTL` | `30d` | 1–365 days, or positive Go duration up to 365 days. |
| `COOKIE_SECURE` | `false` locally | Set true under HTTPS. |
| `ALLOWED_ORIGIN` | Required | One exact browser origin with no trailing slash. |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | Both empty | Optional first-start bootstrap; set both or neither. |
| `PII_ENCRYPTION_KEY` | Required | Base64 of exactly 32 bytes; keep stable. |
| `TRUST_PROXY_HEADERS` | Host false; Docker true | Trust overwritten `X-Real-IP` only behind a private trusted proxy. |
| `API_V1_DEPRECATED` | `false` | Emit v1 Deprecation/Sunset headers. |
| `API_V1_DEPRECATION_DATE`, `API_V1_SUNSET_DATE` | Empty | RFC3339 dates required when deprecation is enabled. |
| `TEST_DATABASE_URL`, `TEST_REDIS_URL` | Unset | Opt into isolated integration checks on the host. |
| `VITE_PROXY_TARGET` | `http://localhost:8080` | Vite development proxy. |
| `VITE_APP_VERSION`, `VITE_APP_COMMIT` | Derived locally/injected by Docker | Public build metadata. |
| `VITE_ADS_ENABLED` | `false` | Optional labelled ad placeholders. |
