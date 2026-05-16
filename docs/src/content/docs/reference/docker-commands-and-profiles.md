---
title: Docker Commands and Profiles
description: Reference supported open-core Docker commands, profiles, and persistent data behavior.
---

Run Docker commands from `open-core/`. If this is your first visit, start with the guided [Quick Start](../start-here/quick-start/) before using this page as a command reference.

Docker startup and provider readiness are separate. Containers can run before credentials are configured, but scouting capabilities depend on the provider and source credentials available in `open-core/.env`.

On macOS and Linux, `pnpm run docker:dev` also best-effort bootstraps the optional local host CLI bridge before Compose starts. The bootstrap creates file-based bridge config and token state, starts `scout-ai-bridge` when possible, writes Docker-facing bridge variables into `open-core/.env` only when the bridge is healthy and has at least one available runtime, and continues to Compose even if the bridge is unavailable. Self-hosted users can also run the bridge alongside a `docker:prod` deployment if they want host-authenticated CLI reuse.

`pnpm run docker:prod` does not run bridge bootstrap, does not require a bridge service, and does not mount bridge token files by default. Use API keys or runtime-local CLI authentication for production AI providers. The bridge is optional for self-hosting, but it is not part of the prod container baseline.

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

For advanced manual bridge control, see [Local AI Bridge](local-ai-bridge/).
