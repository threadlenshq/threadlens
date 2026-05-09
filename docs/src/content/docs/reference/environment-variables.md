---
title: Environment Variables
description: Reference the ThreadLens open-core environment variables without renaming runtime settings.
---

ThreadLens preserves existing runtime environment variable names even when public docs use the ThreadLens brand. Do not rename these variables as part of documentation updates.

Docker reads values from `open-core/.env` unless an embedding repository or command overrides the env file. Docker containers can start before credentials are configured, but scouting capabilities depend on the variables available to the runtime.

| Variable | Default | Capability unlocked | Purpose |
| --- | --- | --- | --- |
| `PORT` | `4749` | Optional runtime override | Go API HTTP port. |
| `SCOUT_DB_PATH` | `../../scout.db` from the API working directory | Optional runtime override | SQLite database path. Docker sets this to `/data/scout.db`. |
| `SCOUT_FRONTEND_DIST` | `../web/dist` from the API working directory | Optional runtime override | Static web build directory served by the Go API. |
| `ANTHROPIC_API_KEY` | Empty | AI scoring, analysis, and reports through Anthropic-backed calls | Anthropic-backed AI workflows. |
| `GEMINI_API_KEY` | Empty | AI scoring, analysis, and reports through Gemini-compatible calls | Gemini provider path. |
| `PARALLEL_API_KEY` | Empty | Google scouting through the configured search provider | Parallel.ai Search provider for Google scouting. |
| `BLUESKY_HANDLE` | Empty | Bluesky scouting | Bluesky API account handle. |
| `BLUESKY_APP_PASSWORD` | Empty | Bluesky scouting | Bluesky app password. |
| `SCOUT_ENV_FILE` | Empty | Optional Docker env-file override | Docker env-file override for embedding repositories. |
| `SCOUT_INIT_DEMO` | Empty | Optional local demo seed | Seeds demo data when set to `1`. |
| `THREADLENS_RUNTIME_MODE` | `self-hosted` | Optional runtime mode selection | Selects `self-hosted` or `hosted`. |

## First-run importance

- Configure at least one AI provider path, such as `ANTHROPIC_API_KEY`, before expecting useful scoring, analysis, or reports.
- Add `PARALLEL_API_KEY` only when you plan to scout Google Search through the configured search provider.
- Add both `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` before relying on Bluesky scouting.
- Leave optional runtime overrides unchanged for the first Docker walkthrough unless you already know you need a custom port, database path, frontend dist path, env file, demo seed, or runtime mode.

Use obviously fake values in docs, examples, and bug reports. Do not commit real provider keys, private URLs, hosted credentials, billing tokens, or customer data.

For a guided setup sequence, see [Configuration Basics](../start-here/configuration-basics/). For Docker command behavior, see [Docker Commands and Profiles](docker-commands-and-profiles/).
