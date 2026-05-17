---
title: Quick Start
description: Start the open-core development profile with Docker from the open-core workspace.
---

Use this path when you want the fastest local ThreadLens setup. Commands in this page run from the `open-core/` directory.

## Prerequisites

- Node and pnpm compatible with `packageManager: pnpm@10.14.0`.
- Docker Desktop or another Docker Engine with Compose support.
- At least one AI provider path before you expect useful AI scoring, analysis, reports, Google scouting, or Bluesky scouting. A provider key is the recommended first-run path.

You can start the app without an AI provider path to smoke-test local startup. Configure a provider key, or use a supported CLI-backed path that is installed and authenticated in the runtime environment, before treating the run as a real scouting workflow.

## Start the development profile

For a first local run, use the development Docker profile:

```bash
pnpm install
pnpm run docker:dev
```

Expected local services:

| Service | URL | Notes |
| --- | --- | --- |
| Go API | `http://localhost:4749` | Uses SQLite stored in the `scout_open_core_sqlite_data` Docker volume. |
| Web app | `http://localhost:4748` | Vite dev server proxies `/api` requests to the Go API container. |

## Verify the app is reachable

Open `http://localhost:4748` in a browser. On a new database, ThreadLens should show the required setup wizard before the main workspace. Complete the wizard in the app to choose an AI provider path and confirm local app/database readiness.

You can also check the API directly:

```bash
curl -i http://localhost:4749/api/onboarding/status
```

Expected result: an HTTP `200` response with JSON fields such as `enabled`, `phase`, `requiredSetupComplete`, and `appDatabase`.

## Before you scout

Docker startup and provider readiness are separate steps. The in-app setup wizard can write supported Docker-mode provider settings to the configured env file, then tell you when a Docker/API restart is needed. You can still configure `.env` manually by following [Configuration Basics](configuration-basics/) if your env file is read-only or you prefer editor-based setup.

Add `PARALLEL_API_KEY` only if you plan to scout Google Search, and add `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` only if you plan to scout Bluesky.

## Fastest first-value path

After setup is complete, follow [First Value in 15 Minutes](first-value-in-15-minutes/) to create one project, add one narrow Reddit query, run one manual scout, and inspect first findings before expanding your query set.

## Start the production self-host profile

After the development profile is verified, you can use the production self-host profile:

```bash
pnpm run docker:prod
```

The production profile serves the built web app from the Go API container at `http://localhost:4749`.

Use the development profile first when you are following the Start Here path because it exposes the web app at `http://localhost:4748` and keeps local debugging straightforward.

## Stop Docker services

```bash
pnpm run docker:down
```

This stops supported `dev` and `prod` profiles without deleting the SQLite data volume. For volume reset behavior and the full command reference, see [Docker Commands and Profiles](../reference/docker-commands-and-profiles/). If the first-run path fails, use [Self-Host Troubleshooting](../reference/self-host-troubleshooting/).
