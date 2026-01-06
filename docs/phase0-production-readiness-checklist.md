# Phase 0 Production Readiness Checklist

## Scope
- Domain and infrastructure hardening
- Auth and calendar sync
- Operational readiness (migrations, backups, health, logging)
- Test coverage for core workflows

## P0: Transactional outbox and reliability
- [x] Wire unit-of-work so aggregate writes + outbox writes are atomic
- [x] Add idempotency to outbox publish path (event ID, publish-once semantics)
- [x] Define retry/backoff policy and dead-letter handling
- [x] Clarify worker topology (standalone worker vs CLI-embedded processor)

## P0: Auth and identity
- [x] Replace hardcoded CLI user with configured identity (`ORBITA_USER_ID`)
- [x] Implement OAuth2 login flow (provider TBD)
- [x] Persist tokens securely (encrypted at rest)
- [x] Add OAuth token refresh monitoring/expiry alerts
- [x] Define encryption key rotation plan for `ORBITA_ENCRYPTION_KEY`

## P0: Calendar sync
- [x] Decide MVP: OAuth sync vs explicit export-only
- [x] If sync: implement provider adapter, token refresh, conflict policy

## P1: Observability and operations
- [x] Structured logging in worker + CLI (context IDs, errors)
- [x] Basic metrics (outbox lag, publish failures, scheduler latency)
- [x] Outbox processor in-process stats (published/failed/dead counts)
- [x] Periodic outbox stats logging in worker
- [x] Outbox lag logged in worker stats
- [x] Health checks for worker/CLI components

## P1: Migrations and data safety
- [x] Migration strategy and rollback plan (documented)
- [x] Backup/restore runbook (documented)
- [x] Outbox cleanup job/schedule in production

## P1: Testing
- [x] ICS export tests
- [x] Auto-schedule workflow tests
- [x] Outbox transaction + publish retry tests
- [x] End-to-end CLI flow with DB
- [x] OAuth integration test (real provider or mocked token exchange)
- [x] Calendar sync integration test (real provider or golden HTTP recordings)

## P1: Calendar enhancements
- [x] Optional delete-on-block-archive behavior
- [x] Calendar reminders/attendees support
- [x] One-way sync back from calendar to Orbita (imports)

## P1: Docs and operations
- [x] Define alert thresholds/SLOs for outbox lag and dead-letter rate
- [x] Incident runbook for OAuth and calendar sync failures
