# open-core/apps/api — Scout Go Backend

This directory contains the Go rewrite of the Scout API. The current production backend remains `apps/api` until the Go service passes Express API parity tests.

## Local development

```bash
pnpm run server:go
```

The Go server listens on `PORT` or `4749` by default and uses repo-root `scout.db` unless `SCOUT_DB_PATH` is set.

## Quality checks

```bash
pnpm run test:go
pnpm run test:api-contract
```

## Architecture

- Chi handlers stay thin and preserve Express response shapes.
- Services hold validation, orchestration, and response shaping.
- SQLite repositories own SQL and transactions.
- Pipelines run in process and use `context.Context` for cancellation.
- AI providers use the same fallback order as Express: Copilot CLI, Claude CLI, Anthropic SDK-compatible HTTP call, Gemini-compatible HTTP call.
