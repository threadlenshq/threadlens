---
title: Environment Variables
description: Complete reference for ThreadLens open-core environment variables, including which capabilities each variable unlocks.
---

ThreadLens preserves existing runtime environment variable names even when public docs use the ThreadLens brand. Do not rename these variables as part of documentation updates.

For first-run setup guidance see [Configuration Basics](../start-here/configuration-basics/). For how these variables are passed to containers see [Docker Commands and Profiles](../reference/docker-commands-and-profiles/).

## Variable reference

| Variable | Default | Capability unlocked | First-run importance |
| --- | --- | --- | --- |
| `PORT` | `4749` | Go API HTTP port. | Not required at first run; override only when the default port conflicts with another service. |
| `SCOUT_DB_PATH` | `../../scout.db` from the API working directory | SQLite database path. Docker sets this to `/data/scout.db`. | Leave unset for standard Docker use; the Docker profile sets this automatically. |
| `SCOUT_FRONTEND_DIST` | `../web/dist` from the API working directory | Static web build directory served by the Go API. | Leave unset for standard Docker use; only needed for custom build locations. |
| `ANTHROPIC_API_KEY` | Empty | Anthropic-backed AI scoring, analysis, and report generation. | **Set this first.** AI scoring and reports do not run without at least one provider key. |
| `GEMINI_API_KEY` | Empty | Gemini provider path for AI scoring and analysis. | Alternative to `ANTHROPIC_API_KEY`; set one or both. |
| `PARALLEL_API_KEY` | Empty | Google scouting through the Parallel.ai Search provider. | Required only when Google scouting is enabled for a project. |
| `BLUESKY_HANDLE` | Empty | Bluesky API account handle for Bluesky scouting. | Required together with `BLUESKY_APP_PASSWORD` when Bluesky scouting is enabled. |
| `BLUESKY_APP_PASSWORD` | Empty | Bluesky app password paired with `BLUESKY_HANDLE`. | Required together with `BLUESKY_HANDLE` when Bluesky scouting is enabled. |
| `SCOUT_ENV_FILE` | Empty | Lets embedding repositories point Docker commands at a different env file. | Not needed for standard first-run use. |
| `SCOUT_INIT_DEMO` | Empty | Seeds demo data when set to `1`. | Optional; useful for evaluating ThreadLens with pre-loaded sample data. |
| `THREADLENS_RUNTIME_MODE` | `self-hosted` | Selects `self-hosted` or `hosted` runtime mode. | Leave at the default `self-hosted` for open-core use. |
