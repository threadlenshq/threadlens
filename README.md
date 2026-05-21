# ThreadLens

ThreadLens is self-hosted research intelligence for finding product opportunities in real conversations across Reddit, Bluesky, and Google Search.

ThreadLens helps builders turn scattered posts, complaints, workarounds, search results, and repeated requests into scored findings and product-angle reports they can inspect locally.

📖 **Full documentation:** [docs.threadlens.dev](https://docs.threadlens.dev)

## Why ThreadLens

Useful market signals are buried across public threads and search results. A founder might see one complaint on Reddit, a workaround on Bluesky, and a related search result days later, but the pattern is easy to miss.

ThreadLens gives you a local-first way to collect those signals, score them for pain and relevance, filter out noise, and generate reports that show recurring themes and possible product angles.

## What it does

- Scouts Reddit, Bluesky, and Google Search from project-specific queries.
- Scores posts and results for pain points, relevance, and opportunity signals.
- Filters duplicates and low-signal results before they become research clutter.
- Generates research reports that cluster findings into pain themes and product angles.
- Builds Google search summaries, opportunities, risks, and next actions.
- Stores projects, runs, posts, reports, schedules, and settings in local SQLite.
- Supports scheduled scout runs for recurring research workflows.
- Runs with Docker or local pnpm commands from this repository workspace.

## Quick start

For most end users, start with the local-first installer:

```bash
npx create-threadlens-app@latest
```

That command creates an isolated local app directory, starts ThreadLens with Docker, checks readiness, and hands you into the setup wizard. Use the raw Docker path below if you are contributing to the repo or want manual control.

### Docker-first path

Run these commands from the repository root.

If you prefer the raw Docker path instead of the pnpm wrappers, use `docker compose` directly with `infra/docker/compose.yml`.

```bash
pnpm install
pnpm run docker:dev
pnpm run self-host:smoke
```

`pnpm run docker:dev` auto-creates `.env` from `.env.example` when it is missing. Add provider keys to `.env` when you want AI analysis, Google Search, or Bluesky features to work. If another repository embeds open-core, it can pass a different env file through `SCOUT_ENV_FILE`.

The development profile starts:

| Service | URL | Notes |
| --- | --- | --- |
| `api` | http://localhost:4749 | Go API with SQLite stored in the `scout_open_core_sqlite_data` Docker volume. |
| `web` | http://localhost:4748 | Vite dev server that proxies `/api` requests to `http://api:4749`. |

After the web app loads, complete the ThreadLens setup wizard, create or accept one starter research project, add one narrow Reddit query, run one scout manually, and inspect the strongest findings before expanding query coverage.

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

Docker usage details live in [`infra/docker/README.md`](infra/docker/README.md). For the shortest activation path after Docker starts, follow [`docs/start-here/first-value-in-15-minutes.md`](docs/src/content/docs/start-here/first-value-in-15-minutes.md).

### Local pnpm checks

Use these commands when you want to validate the Go API without Docker:

```bash
pnpm run test:go
pnpm run build:go
```

## Configuration

ThreadLens reads shared runtime settings from `.env`. Start from the checked-in example:

```bash
cp .env.example .env
```

The Docker commands create that file for you when it is missing. Keep the existing environment variable names because they are runtime configuration, not public branding.

| Variable | Required | Purpose |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | Required when using Anthropic-backed AI workflows | Powers Claude-based analysis and scoring paths. |
| `GEMINI_API_KEY` | Required when using Gemini as the AI provider | Provides the Gemini fallback/provider path. |
| `PARALLEL_API_KEY` | Optional | Enables Parallel.ai Search API for the Google pipeline when configured. |
| `BLUESKY_HANDLE` | Optional | Required for Bluesky scouting. |
| `BLUESKY_APP_PASSWORD` | Optional | Required for Bluesky scouting. |
| `SCOUT_ENV_FILE` | Optional | Lets embedding repositories point Docker commands at a different env file. |

Source-specific integrations are optional. You can start with the providers you have keys for, then add more sources later.

## How it works

1. Create a project for the niche, product idea, or market you want to research.
2. Add project queries for Reddit, Bluesky, or Google Search.
3. Run a scout pass to collect candidate posts and search results.
4. Let ThreadLens score, deduplicate, and filter the findings.
5. Review high-signal posts and Google results in the local web app.
6. Generate reports that cluster pain themes and suggest product angles.
7. Add schedules if you want recurring research runs.

## Architecture

```text
apps/
  web/             Svelte 5 + Vite frontend for the local app
  api/             Go backend and API server
packages/
  shared/          Shared types, constants, and utilities
infra/
  docker/          Docker Compose configuration and Docker usage notes
  docs/              Public documentation and implementation notes
```

The self-hosted app stores data in SQLite and keeps provider keys under your control. The Docker development profile runs the web and API services separately; the production profile serves the built web app through the Go API image.

## Self-Hosted ThreadLens

This repository contains the self-hosted ThreadLens app you can run today. It is designed for builders who want a self-hostable, inspectable research workflow with local data storage.

Hosted ThreadLens is a later possibility, but the current public product and docs are optimized for self-hosted activation first.

## Contributing and status

ThreadLens is early software. The highest-value contributions are setup feedback, reproducible bugs, source-specific pipeline fixes, documentation improvements, and small product-quality improvements that keep the self-hosted workflow simple.

Public product code belongs in this repository workspace. Hosted-only SaaS services, billing, multi-tenancy, and cloud infrastructure belong outside the `open-core/` subtree.

## Releases

Public releases are published to [github.com/threadlenshq/threadlens/releases](https://github.com/threadlenshq/threadlens/releases) from the subtree split of this repository. Each release is tagged at the split SHA corresponding to the monorepo commit being released.

Release notes and version history are maintained in [`CHANGELOG.md`](CHANGELOG.md). Source archives and tarballs are attached to each GitHub release for reproducible builds.

To cut a new release, run from the monorepo root:

```bash
pnpm run release -- --track open-core
```

The maintainer release runbook lives in the private monorepo.

## License status

ThreadLens is source-available under the Business Source License 1.1 in [`open-core/LICENSE`](LICENSE).

The license allows public source access, local modification, redistribution, and self-hosted production use for your own internal business or personal use.

The license does not allow selling ThreadLens as your own product, offering it as a competing hosted service, or using ThreadLens branding without permission.

Public contributions are welcome through GitHub forks and pull requests. See the docs licensing page for the current contribution policy and license scope.

The repository contribution terms are documented in [`CONTRIBUTING.md`](CONTRIBUTING.md).
