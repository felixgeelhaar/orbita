# Phase 1 CLI MVP Checklist

## Scope
- CLI workflows for Orbita Core, Habits, Meetings, and Billing
- Scheduling integrations for tasks, habits, and 1:1s
- Basic adaptive frequency logic
- Entitlement gating for premium modules

## CLI Core
- [x] Task capture via CLI (add/list/complete/archive)
- [x] Daily/weekly planning commands
- [x] Schedule CRUD (add/remove/reschedule/auto)
- [x] Review + stats commands
- [x] End-to-end CLI smoke tests for core workflows

## Smart Habits (CLI)
- [x] Create/list/log/archive habits
- [x] Due-today and streak views
- [x] Habits included in auto-scheduling
- [x] Habit CLI docs in ops/runbook
- [x] Habit completion events wired to adaptive frequency rules

## Smart 1:1 Scheduler (CLI)
- [x] Meetings domain model + repository
- [x] Meeting create/list/update/archive CLI
- [x] Generate upcoming 1:1 candidates (cadence + last-held)
- [x] Schedule integration (meeting blocks added to auto-schedule)
- [x] Tests for meeting cadence + scheduling

## Adaptive Frequency (basic)
- [x] Define cadence adjustment rules (habit + meeting)
- [x] Track completion/attendance signals
- [x] Emit frequency change events
- [x] CLI surfaces new frequency state

## Billing + Entitlements
- [x] Billing domain model (subscription, modules)
- [x] Stripe integration (CLI config + webhook placeholder)
- [x] Entitlement checks for premium modules
- [x] CLI to inspect active plan/modules
- [x] Tests for entitlement gating

## Docs + Ops
- [x] Phase 1 runbook updates (billing + meetings)
- [x] CLI usage examples for meetings and entitlements
