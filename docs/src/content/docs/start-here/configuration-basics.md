---
title: Configuration Basics
description: Configure the provider credentials needed for useful ThreadLens scouting and reports.
---

ThreadLens reads runtime settings from the repository root `.env`. The Docker commands create that file from `.env.example` when it is missing, and you can also create it manually before starting or restarting Docker.

## Fastest useful self-host path

For the first self-hosted run, configure one AI provider key before expecting useful scoring, analysis, or reports. Anthropic through `ANTHROPIC_API_KEY` or Gemini through `GEMINI_API_KEY` is the fastest reliable Docker path. Add `PARALLEL_API_KEY` only for Google Search scouting, and add `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` only for Bluesky scouting.

Local CLI or bridge paths are advanced local options. They are useful when you already have the CLI authenticated in the runtime, but they are not required for the first open-core activation path.

Docker can start without an AI provider path, but useful AI scoring, analysis, reports, Google scouting, and Bluesky scouting need either a configured provider key or a supported CLI-backed provider path available in the same runtime environment as ThreadLens. For the simplest first run, start with one explicit provider key.

## 1. Create or open the environment file

Run from the repository root if you want to create the file before Docker does:

```bash
cp .env.example .env
```

Then open `.env` in your editor and add the AI provider key and source credentials you need for the first source you plan to scout.

## 2. Configure one AI provider first

For a first useful scout, configure at least one AI provider path before expecting scores, analysis, or reports. API keys are the recommended first-run path because they are explicit, easy to verify, and portable across local and Docker runs.

| Provider path | First-run role | Notes |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | Recommended explicit first provider key | Enables Anthropic-backed AI workflows when the runtime uses that provider path. [How&nbsp;to&nbsp;get&nbsp;→](/reference/credential-setup/#anthropic-api-key) |
| `GEMINI_API_KEY` | Alternative AI provider key | Enables the Gemini-compatible provider path when configured. [How&nbsp;to&nbsp;get&nbsp;→](/reference/credential-setup/#gemini-api-key) |
| Copilot CLI | Supported advanced fallback path | Works only when the Copilot CLI is installed and authenticated in the same runtime environment as ThreadLens. [Setup&nbsp;guide&nbsp;→](/reference/credential-setup/#github-copilot-cli) |
| Claude CLI | Supported advanced fallback path | Works only when the Claude CLI is installed and authenticated in the same runtime environment as ThreadLens. [Setup&nbsp;guide&nbsp;→](/reference/credential-setup/#claude-cli) |
| Host CLI bridge | Optional advanced fallback path | Routes AI calls from Docker or a self-hosted runtime to a host-authenticated CLI runtime over a loopback HTTP service. See the bridge guidance below. |

For first-run Docker docs, prefer an explicit provider key because CLI availability and authentication can vary by host and container setup. The CLI-backed paths are supported runtime paths, but this page intentionally does not provide Docker mount, install, or authentication walkthroughs.

### Host CLI bridge (optional fallback)

When ThreadLens runs inside Docker on macOS or Linux and the host machine has Copilot CLI or Claude CLI installed and authenticated, `pnpm run docker:dev` best-effort starts a host helper named `scout-ai-bridge`. Self-hosted operators can also run the same helper alongside a `docker:prod` deployment if they want to reuse a host-authenticated CLI runtime. The helper lets the Dockerized API route CLI-backed AI calls to the host without mounting CLI credential directories into the container.

For most users, that automatic Docker bootstrap is the only bridge setup needed. The Docker bootstrap now enables bridge env vars only when the bridge is actually healthy and has at least one available runtime.

The bridge is still a fallback path. API keys remain the recommended explicit first-run choice because they are easier to verify and work the same inside and outside Docker. For self-hosters, the bridge is optional convenience, not a requirement.

**What Docker dev creates:**
- `$XDG_CONFIG_HOME/scout/ai-bridge.token`, or `~/.config/scout/ai-bridge.token`, containing a local bearer token.
- `$XDG_CONFIG_HOME/scout/ai-bridge.json`, or `~/.config/scout/ai-bridge.json`, pointing host-side clients at `http://127.0.0.1:4761`.
- `.env` bridge values that point the API container at `http://host.docker.internal:4761` and mount the token file read-only at `/run/secrets/scout-ai-bridge-token`.

**Network safety:**
- The helper binds to `127.0.0.1:4761` by default and rejects wildcard or public bind addresses.
- Exposing the bridge publicly is unsafe and unsupported.
- From inside Docker, `localhost` refers to the container itself, so the Docker-facing URL uses `host.docker.internal` instead.

**Opt out:**
- Set `SCOUT_AI_BRIDGE_DISABLE=1` before running `pnpm run docker:dev` to skip bridge bootstrap.
- Bridge bootstrap failures do not block Docker startup. If the helper cannot build, launch, or find an authenticated CLI, ThreadLens falls back to direct in-container providers and configured API keys.

**Advanced manual control:**
- Run `pnpm run bridge:start` from the repository root to start the bridge manually.
- Run `pnpm run bridge:status` or `pnpm run bridge:health` to verify host CLI availability.
- Run `pnpm run bridge:stop` to stop the managed bridge process.
- See [Local AI Bridge](../reference/local-ai-bridge/) for details.

**UI status:**
- The Models UI shows bridge status as a read-only indicator. It is not a separate model mode or model picker.

## 3. Add source-specific credentials only when needed

AI provider configuration is separate from source access. Add source credentials for the source you want to scout:

| Source | Credentials | First-run guidance |
| --- | --- | --- |
| Reddit | No extra source credential is documented for the current first-run path | Lowest-friction first source, but still needs an AI provider for useful scoring and reports. |
| Google Search | `PARALLEL_API_KEY` | Add this when you want Google scouting through the configured search provider. [How&nbsp;to&nbsp;get&nbsp;→](/reference/credential-setup/#parallel-api-key) |
| Bluesky | `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` | Add both before relying on Bluesky scouting. [How&nbsp;to&nbsp;get&nbsp;→](/reference/credential-setup/#bluesky-credentials) |

## Restart Docker after editing `.env`

Save `.env`, then restart the Docker profile you started so the containers pick up the new values:

```bash
# From the repo root
pnpm run docker:down
pnpm run docker:dev   # or pnpm run docker:prod if you started with the prod profile
```

Environment variables are read at container start-up, so any change to `.env` requires a restart before ThreadLens sees the new values.

## 4. Leave optional runtime overrides alone at first

These variables are useful for advanced local development, embedding repositories, or deployment customization, but they are not required for a first scout:

| Variable | Purpose |
| --- | --- |
| `SCOUT_ENV_FILE` | Lets embedding repositories point Docker commands at a different env file. |
| `SCOUT_DB_PATH` | Overrides the SQLite database path used by the Go API. |
| `SCOUT_FRONTEND_DIST` | Points the Go API at a built web app directory for static serving. |
| `SCOUT_INIT_DEMO` | Seeds demo data when set to `1`. |
| `THREADLENS_RUNTIME_MODE` | Selects self-hosted or hosted runtime mode; self-hosted is the default for this setup. |

## Onboarding flow and first-run persistence

ThreadLens shows a guided onboarding flow to new users on first launch. The blocking required setup covers provider selection and local app/database readiness. After required setup, the normal workspace opens and an optional checklist can help you create a starter project, add a query, run a scout, review findings, open reports, and visit Models/Settings.

Onboarding progress is stored server-side in the app database, so refreshes and browser changes resume from the backend state. In Docker mode, supported configuration values are written through the backend to the configured env file. Environment variables are still read by the running API process at startup, so provider or source changes may require restarting Docker before they affect AI calls.

Three variables control first-run behavior:

| Variable | Purpose |
| --- | --- |
| `SCOUT_ONBOARDING_MODE` | Set to `docker` in container environments. When set, the wizard writes supported completed configuration back to `/data/.env` or the configured onboarding env file so it persists across container restarts. Leave unset for non-containerised installs. |
| `SCOUT_ONBOARDING_ENV_FILE` | Overrides the Docker-mode onboarding env-file write target. Use this only when the container has a writable mounted env file at a different path. |
| `SCOUT_ONBOARDING_DISABLE` | Set to `1` to skip the required wizard and optional checklist entirely. Use this for automated or pre-configured deployments where all required env vars are already present. |

The Docker Compose files in this repository set `SCOUT_ONBOARDING_MODE=docker` automatically, so the default Docker path handles persistence without manual configuration.

## Safe sample values

Use obviously fake values in examples, screenshots, and bug reports:

```dotenv
ANTHROPIC_API_KEY=sk-ant-example-not-real
GEMINI_API_KEY=gemini-example-not-real
PARALLEL_API_KEY=parallel-example-not-real
BLUESKY_HANDLE=example.bsky.social
BLUESKY_APP_PASSWORD=example-app-password-not-real
```

Do not commit real provider keys, private URLs, hosted credentials, billing tokens, or customer data.

## Next steps

- Continue to [Create Your First Project](first-project/) after Docker is running and at least one AI provider path is configured. For most first runs, use a provider key.
- See [Credential Setup](/reference/credential-setup/) for step-by-step instructions on obtaining each credential.
- See [Model and Provider Configuration](../user-guide/model-provider-configuration/) for provider fallback behavior.
- See [AI Providers](../architecture/ai-providers/) for the architecture-level provider order.
- See [Environment Variables](../reference/environment-variables/) for the complete variable reference.
