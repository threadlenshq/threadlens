---
title: ---
description: Understand the active open-core backend structure and responsibilities.
---


The active open-core backend lives in `open-core/apps/api`.

## Responsibility split

- Chi handlers accept HTTP requests and preserve response shapes used by the frontend.
- Services hold validation, orchestration, response shaping, and pipeline coordination.
- SQLite repositories own SQL and transactions.
- Pipelines run in process and use `context.Context` for cancellation.
- App configuration reads runtime settings such as `PORT`, `SCOUT_DB_PATH`, and `SCOUT_FRONTEND_DIST`.

## Default runtime

The Go API listens on `4749` when `PORT` is unset. In production Docker, it also serves the built web app from `SCOUT_FRONTEND_DIST=/app/web/dist`.
