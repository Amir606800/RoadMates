# Changelog

All notable changes are recorded here using [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Account registration, login, rotating refresh sessions with reuse detection,
  logout, current-user endpoint, role middleware, and admin bootstrap.
- React sign-in/registration, memory-only access tokens, session restoration,
  proactive refresh, and role guards backed by server-issued roles.
- Authentication security and PostgreSQL integration checks.
- Driver applications with AES-GCM encrypted PINs, admin review screens,
  transactional approval/role grants, rejection/resubmission, and automatic
  applicant role refresh after approval.
- Encryption, access-control, concurrency, and approval rollback checks.
- Public, database-backed district search with Azerbaijani letter folding and
  searchable mobile pickers with permit hints in client and driver screens.
- Ride publishing, public search and details, owner cancellation/completion,
  and one active ride per driver enforced by service checks and PostgreSQL.
- Live ride cards and driver controls, Azerbaijan departure dates, precise
  monetary validation, and concurrent publish/search integration checks.
- Redis cache-aside district/search caching with singleflight, TTL jitter,
  namespace invalidation/recovery, and ID-only ride results to exclude personal data.
- Shared Redis fixed-window rate limits with atomic expiry and a bounded
  per-process outage fallback.
- Isolated PostgreSQL/Redis test containers, Go race checks, cache/limiter and
  operational coverage, validated OpenAPI v1, and complete setup/versioning docs.

## [0.1.0] - 2026-10-23

### Added

- Container setup for the backend, frontend, PostgreSQL, Redis, and migrations.
- Versioned API skeleton, operational endpoints, request tracing, and app compatibility checks.
- Initial database schema and all 63 Azerbaijani districts.
- App build metadata, frontend version footer, and local release commands.
