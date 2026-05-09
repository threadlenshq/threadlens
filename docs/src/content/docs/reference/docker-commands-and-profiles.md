---
title: Docker Commands and Profiles
description: Reference supported open-core Docker commands, profiles, and persistent data behavior.
---

Run Docker commands from `open-core/`. If this is your first visit, start with the guided [Quick Start](../start-here/quick-start/) before using this page as a command reference.

Docker startup and provider readiness are separate. Containers can run before credentials are configured, but scouting capabilities depend on the provider and source credentials available in `open-core/.env`.

| Command | Profile | Result |
| --- | --- | --- |
| `pnpm run docker:dev` | `dev` | Starts `api` and `web` services for local development. The web app is available at `http://localhost:4748`, and the Go API is available at `http://localhost:4749`. |
| `pnpm run docker:prod` | `prod` | Starts `api-prod`, which serves the built web app through the Go API at `http://localhost:4749`. |
| `pnpm run docker:down` | `dev` and `prod` | Stops supported profiles without deleting SQLite data. |

## Persistent data

SQLite data is stored in the named Docker volume `scout_open_core_sqlite_data` and mounted at `/data/scout.db` inside API containers.

To intentionally delete local Docker data, run:

```bash
docker volume rm scout_open_core_sqlite_data
```

Only run the reset command when you want to delete local ThreadLens data.

For credential setup after Docker starts, see [Configuration Basics](../start-here/configuration-basics/).
