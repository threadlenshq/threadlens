# Scout Open-Core

Scout Open-Core is the self-hostable Scout application. It contains the Go API, Svelte/Vite web app, shared packages, and Docker assets needed to run Scout without the private SaaS repository.

## Docker quick start

Run these commands from the `open-core/` directory:

```bash
pnpm install
cp .env.example .env
pnpm run docker:dev
```

The development profile starts:

| Service | URL | Notes |
| --- | --- | --- |
| `api` | http://localhost:4749 | Go API with SQLite stored in the `scout_open_core_sqlite_data` Docker volume. |
| `web` | http://localhost:4748 | Vite dev server that proxies `/api` requests to `http://api:4749`. |

For the production/self-host profile:

```bash
pnpm run docker:prod
```

The production profile starts `api-prod` on http://localhost:4749. The Go API image includes the built web app and serves it through `SCOUT_FRONTEND_DIST=/app/web/dist`.

Stop either profile with:

```bash
pnpm run docker:down
```

`docker:down` stops the supported `dev` and `prod` profiles without deleting the SQLite data volume.

## Local non-Docker checks

```bash
pnpm run test:go
pnpm run build:go
```

Docker usage details live in `infra/docker/README.md`.
