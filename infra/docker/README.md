# Open-Core Docker

`compose.yml` is the canonical Docker Compose entry point for Scout Open-Core. The public command surface is intentionally short:

```bash
pnpm run docker:dev
pnpm run docker:prod
pnpm run docker:down
```

The Docker env file lives at `open-core/.env` by default. If it is missing, `pnpm run docker:dev` and `pnpm run docker:prod` create it from `open-core/.env.example` before starting Compose. Embedding repos can override the env location by setting `SCOUT_ENV_FILE`.

## Profiles

| Profile | Command | Services | Ports |
| --- | --- | --- | --- |
| `dev` | `pnpm run docker:dev` | `api`, `web` | API: 4749, Web: 4748 |
| `prod` | `pnpm run docker:prod` | `api-prod` | API + Web: 4749 |

The `dev` profile builds both images and starts the Go API plus Vite web app. The web container sets `VITE_API_PROXY_TARGET=http://api:4749` so browser `/api` requests reach the Go API through Vite.

The `prod` profile builds the web app inside the Go API image and serves the resulting static files from `/app/web/dist` through the Go server. No private-root services are required.

## Persistent data

SQLite data is stored in the named Docker volume `scout_open_core_sqlite_data` and mounted at `/data/scout.db` inside API containers. `pnpm run docker:down` stops containers but does not remove this volume.

To intentionally reset local Docker data, run:

```bash
docker volume rm scout_open_core_sqlite_data
```

Only run the reset command when you want to delete local Scout data.
