---
title: Quick Start
description: Start the open-core development profile with Docker from the open-core workspace.
---


Use this path when you want the fastest local ThreadLens setup. Commands in this page run from the `open-core/` directory.

## Prerequisites

- Node and pnpm compatible with `packageManager: pnpm@10.14.0`.
- Docker Desktop or another Docker Engine with Compose support.
- Provider keys if you want AI analysis, Google scouting, or Bluesky scouting.

## Start the development profile

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

## Start the production self-host profile

```bash
pnpm run docker:prod
```

The production profile serves the built web app from the Go API container at `http://localhost:4749`.

## Stop Docker services

```bash
pnpm run docker:down
```

This stops supported `dev` and `prod` profiles without deleting the SQLite data volume.
