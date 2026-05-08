# open-core/apps/api — Scout Go Backend

This directory contains the Go backend for Scout. It is the active open-core API. The legacy Express backend has been moved to `apps/legacy-express` and is parked.

📖 **Architecture docs:** [docs.threadlens.dev/architecture/go-api/](https://docs.threadlens.dev/architecture/go-api/)

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
