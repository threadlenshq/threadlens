---
title: ---
description: Reference the ThreadLens open-core environment variables without renaming runtime settings.
---


ThreadLens preserves existing runtime environment variable names even when public docs use the ThreadLens brand.

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `4749` | Go API HTTP port. |
| `SCOUT_DB_PATH` | `../../scout.db` from the API working directory | SQLite database path. Docker sets this to `/data/scout.db`. |
| `SCOUT_FRONTEND_DIST` | `../web/dist` from the API working directory | Static web build directory served by the Go API. |
| `ANTHROPIC_API_KEY` | Empty | Anthropic-backed AI workflows. |
| `GEMINI_API_KEY` | Empty | Gemini provider path. |
| `PARALLEL_API_KEY` | Empty | Parallel.ai Search provider for Google scouting. |
| `BLUESKY_HANDLE` | Empty | Bluesky API account handle. |
| `BLUESKY_APP_PASSWORD` | Empty | Bluesky app password. |
| `SCOUT_ENV_FILE` | Empty | Docker env-file override for embedding repositories. |
| `SCOUT_INIT_DEMO` | Empty | Seeds demo data when set to `1`. |
| `THREADLENS_RUNTIME_MODE` | `self-hosted` | Selects `self-hosted` or `hosted`. |

Do not rename these variables as part of documentation updates.
