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

Open `http://localhost:4748` in a browser. The app should load the ThreadLens interface and call the API through the Vite proxy.

You can also check the API directly:

```bash
curl -i http://localhost:4749/api/runtime/capabilities
```

Expected result: an HTTP `200` response with JSON runtime information.

## Before you scout

Docker startup and provider readiness are separate steps. A no-key launch confirms the containers, web app, and API can run, but it does not provide a complete first-scout outcome.

Before creating a real first scout, follow [Configuration Basics](configuration-basics/) to configure at least one AI provider path. For most first runs, use a provider key. Add `PARALLEL_API_KEY` only if you plan to scout Google Search, and add `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` only if you plan to scout Bluesky.

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

This stops supported `dev` and `prod` profiles without deleting the SQLite data volume. For volume reset behavior and the full command reference, see [Docker Commands and Profiles](../reference/docker-commands-and-profiles/).
