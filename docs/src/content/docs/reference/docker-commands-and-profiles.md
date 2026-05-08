---
title: Docker Commands and Profiles
description: Reference supported open-core Docker commands, profiles, and persistent data behavior.
---


Run Docker commands from `open-core/`.

| Command | Profile | Result |
| --- | --- | --- |
| `pnpm run docker:dev` | `dev` | Starts `api` and `web` services for local development. |
| `pnpm run docker:prod` | `prod` | Starts `api-prod`, which serves the built web app through the Go API. |
| `pnpm run docker:down` | `dev` and `prod` | Stops supported profiles without deleting SQLite data. |

## Persistent data

SQLite data is stored in the named Docker volume `scout_open_core_sqlite_data` and mounted at `/data/scout.db` inside API containers.

To intentionally delete local Docker data, run:

```bash
docker volume rm scout_open_core_sqlite_data
```

Only run the reset command when you want to delete local ThreadLens data.
