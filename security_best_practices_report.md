# Security and Feature Audit Report

Date: 2026-02-18
Scope: `/Users/davidojo/Desktop/go-projects/uas-go`

## Executive Summary
The application is substantially closer to production readiness and now passes `go test ./...`, `go build ./...`, and `go vet ./...`.

Major security remediations are in place (JWT validation hardening, secure OTP generation, refresh-token session binding/rotation, token hashing at rest, stricter CORS behavior, and audit-log token redaction).

Remaining issues are mostly operational/completeness gaps plus a few medium-risk security hardening items.

## Feature Audit Status
1. Email verification: **Implemented**.
2. Password reset: **Implemented**.
3. Rate limiting: **Implemented with limitations** (in-memory, per-instance global limiter).
4. OTP login: **Implemented**.
5. Magic link login: **Implemented**.
6. RBAC: **Implemented** (admin enforcement on sensitive routes present).
7. Audit logging: **Implemented with limitations** (retention service not scheduled/wired).
8. Deployment/migrations: **Not fully complete** (SQL migration schema and runtime model schema are divergent).

## Remaining Findings

### High

#### F-001: Migration schema drift can break clean deployments
- Rule ID: GO-DEPLOY-001 / completeness
- Severity: High
- Location: `migrations/sql/0001_create_users_table.up.sql:6`, `cmd/api/main.go:39`, `internal/repositories/user_repository.go:23`
- Evidence:
  - Migration creates `users` with columns like `password_hash`, `first_name`, `last_name`.
  - Runtime repositories use GORM models (`UserModel`) and app now relies on `AutoMigrate` at startup.
- Impact: Environments depending on migration SQL as source-of-truth can drift from runtime schema, causing inconsistent behavior or data split across unexpected tables.
- Fix: Align SQL migrations with current GORM models and make one strategy authoritative (migration-first preferred for production).
- Mitigation: Keep startup `AutoMigrate` as temporary safety net until SQL migration set is fully aligned.

### Medium

#### F-002: Panic-prone context assertions (DoS risk on missing context keys)
- Rule ID: GO-CONC-001 / robustness
- Severity: Medium
- Location: `internal/helpers/ctx_helper.go:26`, `internal/helpers/error_helper.go:25`, `internal/helpers/error_helper.go:47`, `internal/helpers/error_helper.go:88`
- Evidence: direct type assertions like `.(string)` on context values without nil/type checks.
- Impact: If middleware ordering changes or handlers are reused without trace context, requests can panic.
- Fix: Replace direct assertions with checked extraction (`v, ok := ...`) and fallback defaults.
- Mitigation: Keep trace middleware globally mounted before all handlers.

#### F-003: Global rate limiter is per-process, not shared across instances
- Rule ID: GO-HTTP-007 / anti-abuse
- Severity: Medium
- Location: `internal/middleware/rate_limit_middleware.go:15`, `internal/middleware/rate_limit_middleware.go:33`
- Evidence: in-memory `clients map[string]*ClientInfo` storage.
- Impact: In multi-instance deployments, clients can bypass effective limits by spreading requests across nodes.
- Fix: Move global limiter state to a shared backend (Redis) or edge/API gateway.
- Mitigation: Keep endpoint-level Redis limiter enabled for high-risk auth flows.

#### F-004: Audit retention service exists but is not wired/scheduled
- Rule ID: Operational completeness
- Severity: Medium
- Location: `internal/services/retention_service.go:17`
- Evidence: retention service methods exist, but no runtime wiring from `cmd/api/main.go`.
- Impact: Audit tables can grow without policy enforcement, increasing storage/cost and query degradation over time.
- Fix: Add a scheduled cleanup job (cron/worker) calling `CleanupOldAuditLogs`.
- Mitigation: Run manual cleanup against `audit_logs` on an operational schedule.

## Resolved in This Pass (Highlights)
1. CORS no longer defaults to wildcard origin (`internal/middleware/security_headers_middleware.go:38`).
2. Audit logs now redact sensitive query params and no longer async-spawn per request (`internal/middleware/audit_middleware.go:97`, `internal/middleware/audit_middleware.go:116`).
3. Refresh token flow now validates server-side session binding and rotates sessions (`internal/handlers/user_handler.go:659`, `internal/handlers/user_handler.go:687`).
4. Refresh/session tokens are hashed at rest with legacy compatibility fallback (`internal/repositories/session_repository.go:29`, `internal/repositories/session_repository.go:35`).
5. Reset/magic auth tokens are hashed at rest with legacy compatibility fallback (`internal/repositories/auth_repository.go:26`, `internal/repositories/auth_repository.go:45`).
6. Audit context extraction now uses actual context helpers (user/department IDs captured correctly) (`internal/middleware/audit_middleware.go:270`).
7. Rate limiter and security audit IP extraction no longer trusts spoofable proxy headers by default (`internal/middleware/rate_limit_middleware.go:83`, `internal/helpers/security_logger_helper.go:46`).
8. Build server schema bootstrap now includes all current models via `AutoMigrate` (`cmd/api/main.go:39`).

