---
title: ---
description: Configure ThreadLens environment variables safely without renaming runtime settings.
---


ThreadLens reads open-core runtime settings from `open-core/.env`. The Docker commands create that file from `open-core/.env.example` when it is missing.

## Create the environment file manually

Run from `open-core/` if you want to create the file before Docker does:

```bash
cp .env.example .env
```

## Supported variables

| Variable | Required | Purpose |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | Required for Anthropic-backed AI workflows | Powers Claude-based analysis and scoring paths. |
| `GEMINI_API_KEY` | Required when using Gemini as the AI provider | Provides the Gemini fallback/provider path. |
| `PARALLEL_API_KEY` | Required for Google scouting when using Parallel.ai Search | Enables the Google search provider path. |
| `BLUESKY_HANDLE` | Required for Bluesky scouting | Identifies the Bluesky account used for API access. |
| `BLUESKY_APP_PASSWORD` | Required for Bluesky scouting | Authenticates the Bluesky account. |
| `SCOUT_ENV_FILE` | Optional | Lets embedding repositories point Docker commands at a different env file. |
| `SCOUT_DB_PATH` | Optional | Overrides the SQLite database path used by the Go API. |
| `SCOUT_FRONTEND_DIST` | Optional | Points the Go API at a built web app directory for static serving. |
| `SCOUT_INIT_DEMO` | Optional | Seeds demo data when set to `1`. |
| `THREADLENS_RUNTIME_MODE` | Optional | Selects `self_hosted` or `hosted`; self-hosted is the default. |

## Safe sample values

Use obviously fake values in examples, screenshots, and bug reports:

```dotenv
ANTHROPIC_API_KEY=sk-ant-example-not-real
GEMINI_API_KEY=gemini-example-not-real
PARALLEL_API_KEY=parallel-example-not-real
BLUESKY_HANDLE=example.bsky.social
BLUESKY_APP_PASSWORD=example-app-password-not-real
```

Do not commit real provider keys, private URLs, hosted credentials, billing tokens, or customer data.
