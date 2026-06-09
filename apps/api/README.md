# open-core/apps/api — Scout Go Backend

This directory contains the Go backend for Scout. It is the active open-core API.

📖 **Architecture docs:** [docs.threadlens.dev/architecture/go-api/](https://docs.threadlens.dev/architecture/go-api/)

## Local development

```bash
pnpm run server:go
```

The Go server listens on `PORT` or `4749` by default and uses repo-root `scout.db` unless `SCOUT_DB_PATH` is set.

## Quality checks

```bash
pnpm run test:go
```

## Architecture

- Chi handlers stay thin.
- Services hold validation, orchestration, and response shaping.
- SQLite repositories own SQL and transactions.
- Pipelines run in process and use `context.Context` for cancellation.
- AI providers: OpencodeProvider (in-process), OpencodeRuntime (bridge), Copilot CLI, Claude CLI, Anthropic SDK, Gemini SDK. Fallback order: `copilot:gpt-5-mini` → `opencode-go:deepseek-v4-flash` → `claude-cli:haiku` → `sdk:haiku` → `gemini:2.5-flash`.
