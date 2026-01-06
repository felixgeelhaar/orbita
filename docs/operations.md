# Operations Runbook

## Scope
This runbook covers Phase 0 operational readiness: migrations, rollbacks, backups, health checks, and outbox cleanup.

## Environments
- Development: local Docker or direct services
- Production: managed PostgreSQL + RabbitMQ

## Environment Variables (minimum)
- `DATABASE_URL`
- `RABBITMQ_URL`
- `ORBITA_ENCRYPTION_KEY`
- `OUTBOX_POLL_INTERVAL`
- `OUTBOX_BATCH_SIZE`
- `OUTBOX_MAX_RETRIES`
- `OUTBOX_STATS_INTERVAL`
- `OUTBOX_RETENTION_DAYS`
- `OUTBOX_CLEANUP_INTERVAL`
- `OUTBOX_PROCESSOR_ENABLED`
- `WORKER_HEALTH_ADDR`
- `ORBITA_USER_ID`
- `OAUTH_CLIENT_ID`
- `OAUTH_CLIENT_SECRET`
- `OAUTH_AUTH_URL`
- `OAUTH_TOKEN_URL`
- `OAUTH_REDIRECT_URL`
- `OAUTH_SCOPES`
- `OAUTH_PROVIDER` (set to `google` for calendar sync)
- `CALENDAR_DELETE_MISSING`
- `CALENDAR_ID`
- `STRIPE_API_KEY`
- `STRIPE_WEBHOOK_SECRET`
- `MCP_ADDR`
- `MCP_AUTH_TOKEN`

## Health Checks
- Worker:
  - `GET /healthz` on `WORKER_HEALTH_ADDR` (includes outbox stats)
  - `GET /readyz` on `WORKER_HEALTH_ADDR` (DB ping)
- CLI:
  - `orbita health`

## Migrations
### Apply
1) Ensure `DATABASE_URL` is set.
2) Run your migration tool against the `migrations/` directory (up).

### Rollback
1) Identify the last applied migration.
2) Run the migration tool to step down (down) to the prior migration.
3) Verify app health via worker `/readyz`.

## Backups & Restore (Postgres)
### Backup
- Schedule daily logical backups of the primary database.
- Keep at least 7 days of backups, with weekly retention for 4 weeks.

### Restore
1) Create a new database instance.
2) Restore the latest logical backup.
3) Apply any missing migrations.
4) Point services to the restored `DATABASE_URL`.
5) Validate with worker `/readyz`.

## Outbox Cleanup
- The outbox table retains published messages for `olderThanDays`.
- The worker runs a cleanup job on `OUTBOX_CLEANUP_INTERVAL`.
- Retention is controlled by `OUTBOX_RETENTION_DAYS`.
- Suggested retention: 7–30 days for production.

## Operational Checks
- Worker log lines:
  - `outbox stats` includes `published`, `failed`, `dead`, `lag_seconds`.
- Alerts to consider:
  - `dead` increasing
  - `lag_seconds` continuously rising
  - `readyz` returning non-200

## Worker Topology
- Production uses a standalone worker (`cmd/worker`) for outbox processing.
- CLI can disable its internal processor via `OUTBOX_PROCESSOR_ENABLED=false`.

## MCP Server
- Start with `orbita mcp serve` (uses the shared MCP runner).
- Run `make mcp-serve` to rebuild the CLI binary and start the MCP server in one step.
- Build with `make build-mcp` or run `go run ./cmd/mcp`.
- Configure `MCP_ADDR` (default `0.0.0.0:8082`).
- Set `MCP_AUTH_TOKEN` and pass it as `Authorization: Bearer <token>`.
- MCP transport uses HTTP + SSE.
- Endpoints:
  - `POST /mcp` (JSON-RPC)
  - `GET /mcp/sse` (server-sent events)
  - `GET /health`

## OAuth Token Monitoring
- `orbita sync` logs a warning when OAuth tokens are near expiry.
- Token refresh errors are logged as warnings and fail the sync.

## Key Rotation
- Rotate `ORBITA_ENCRYPTION_KEY` by re-encrypting stored tokens:
  1) Generate a new base64 32-byte key.
  2) Add the new key and temporarily keep the old key available for decryption.
  3) Re-encrypt all stored tokens with the new key.
  4) Remove the old key.

## SLOs and Alerts
- Outbox lag: alert if `lag_seconds` > 60s for 5 minutes.
- Dead-letter rate: alert if `dead` increases consistently over 10 minutes.
- Calendar sync failures: alert if sync failures > 5% over 1 hour.

## Incident Runbook (OAuth/Calendar Sync)
- Symptoms: sync fails, token refresh errors, OAuth HTTP errors.
- Checks:
  - Confirm `OAUTH_*` env vars and `ORBITA_ENCRYPTION_KEY`.
  - Check worker logs for `oauth token refresh failed`.
  - Validate provider status page.
- Actions:
  - Re-run `orbita auth url` and re-authorize.
  - Rotate keys if decrypt errors appear after config changes.
  - Disable delete-missing if deletions are suspected.

## Calendar Sync
### Setup
- Configure OAuth for Google Calendar (`OAUTH_PROVIDER=google`).
- Run `orbita auth url` and complete the OAuth flow.
- Store tokens with `orbita auth exchange --code <code>`.

### Sync
- Run `orbita sync --days 7` to sync the next 7 days of blocks to the primary calendar.
- Use `orbita sync --days 7 --delete-missing` to delete remote events that are not present in the current sync set.
- Use `orbita sync --calendar <id>` to target a specific calendar ID.
- Use `orbita sync --use-config-calendar=false` to ignore `CALENDAR_ID` and target `primary`.
- Use `orbita sync --attendee person@example.com` (repeatable) to add attendees to synced events.
- Use `orbita sync --reminder 10 --reminder 30` to add reminder overrides in minutes.
- Use `orbita settings calendar set --calendar <id>` to store a per-user calendar ID.
- Use `orbita settings calendar delete-missing set --value=true` to store delete-missing preference.
- Use `orbita settings calendar list` to list available calendars.
- Use `orbita settings calendar list --primary-only` to show only the primary calendar.
- Use `orbita settings calendar list --json` for machine-readable output.
- Use `orbita settings calendar get --json` or `orbita settings calendar delete-missing get --json` for JSON output.
- Use `orbita settings calendar set --calendar <id> --json` or `orbita settings calendar delete-missing set --value=true --json` for JSON output.

### Import
- Run `orbita schedule import --days 7 --type meeting` to import events as schedule blocks.
- Use `orbita schedule import --tagged-only` to only import events created by Orbita.
- Use `orbita schedule import --calendar <id>` to import from a specific calendar.
- Use `orbita schedule import --use-config-calendar=false` to ignore `CALENDAR_ID` and use `primary`.

### Limitations
- Sync targets the primary Google Calendar by default; use `--calendar` or `CALENDAR_ID` to change it.
- Events are upserted by Orbita block ID; deletion is optional via `--delete-missing`.
- Attendees and reminders are only added when configured in `orbita sync`.
- When `delete-missing` is enabled, `schedule remove` attempts to delete the calendar event directly.

## Habits
- Create a habit with `orbita habit create "Morning review" --frequency daily --duration 15`.
- List habits with `orbita habit list` or `orbita habit list --due`.
- Log completion with `orbita habit log <habit-id>` or `orbita done <prefix>`.
- Archive a habit with `orbita habit archive <habit-id>`.
- Run `orbita adapt --habits` to adjust habit frequency based on recent completions.

## Meetings
- Create a 1:1 with `orbita meeting create "Alex" --cadence weekly --duration 30 --time 10:00`.
- List meetings with `orbita meeting list` (use `--archived` for all).
- Update a meeting with `orbita meeting update <meeting-id> --cadence biweekly`.
- Mark a meeting held with `orbita meeting held <meeting-id> --date 2024-02-02 --time 09:30`.
- Archive a meeting with `orbita meeting archive <meeting-id>`.
- Run `orbita adapt --meetings` to adjust meeting cadence based on attendance.
- Use `orbita schedule auto --meetings` to include 1:1 candidates in auto-scheduling.

## Billing
- Check subscription with `orbita billing status`.
- List entitlements with `orbita billing entitlements`.
- Grant a module with `orbita billing grant --module adaptive-frequency --active`.
- Process a webhook payload with `orbita billing webhook --event ./event.json`.
