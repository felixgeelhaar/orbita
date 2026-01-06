# AGENTS.md

## Purpose
Shared guidance for contributors and automation in this repo.

## Workflow
- Prefer `rg` for searching.
- Use `gofmt` for Go formatting.
- Keep edits ASCII unless the file already uses Unicode.
- Avoid destructive git commands unless explicitly requested.

## Testing
- Run focused tests for touched packages when possible.
- If `TEST_DATABASE_URL` is not set, integration tests should skip.

## Output
- Keep logs and CLI output deterministic and structured where possible.
- Prefer `cmd.OutOrStdout()` for CLI command output.
