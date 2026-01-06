# Phase 2 Intelligence & Power Checklist

## Scope
- Auto-Rescheduler for conflicts and missed blocks
- Priority Engine Pro for scoring and ranking work
- AI Inbox Pro for capture + classification
- Automations Pro for triggers/actions
- MCP v1 interface adapter

## Auto-Rescheduler
- [x] Define reschedule policy (missed blocks, conflicts, overdue tasks)
- [x] Persist reschedule attempts and outcomes
- [x] Auto-reschedule command and background worker flow
- [x] Conflict resolution rules (priority + due date)
- [x] Tests for reschedule scenarios

## Priority Engine Pro
- [ ] Priority score model (signals: due date, effort, streak risk, meeting cadence)
- [ ] Store computed score and explanation
- [ ] CLI command to recalc priorities
- [ ] Integrate scheduler with priority scores
- [ ] Tests for scoring + ordering

## AI Inbox Pro
- [ ] Inbox domain model + persistence
- [ ] CLI capture command (text + metadata)
- [ ] Classification pipeline (rules + AI placeholder)
- [ ] Promote inbox items to tasks/habits/meetings
- [ ] Tests for ingestion and promotion

## Automations Pro
- [ ] Automation rule model (trigger + conditions + action)
- [ ] CLI for managing automations
- [ ] Event-driven execution (outbox consumer)
- [ ] Action handlers (create task, reschedule, notify)
- [ ] Tests for trigger evaluation

## MCP v1
- [x] MCP adapter scaffolding + tool registry
- [x] Authenticated MCP entry point
- [x] Tool mapping to application services
- [x] Tests for MCP tool calls

## Docs + Ops
- [x] Phase 2 runbook updates
- [x] CLI examples for new features
