---
title: Configuration Basics
description: Configure the provider credentials needed for useful ThreadLens scouting and reports.
---

ThreadLens reads open-core runtime settings from `open-core/.env`. The Docker commands create that file from `open-core/.env.example` when it is missing, and you can also create it manually before starting or restarting Docker.

Docker can start without an AI provider path, but useful AI scoring, analysis, reports, Google scouting, and Bluesky scouting need either a configured provider key or a supported CLI-backed provider path available in the same runtime environment as ThreadLens. For the simplest first run, start with one explicit provider key.

## 1. Create or open the environment file

Run from `open-core/` if you want to create the file before Docker does:

```bash
cp .env.example .env
```

Then open `open-core/.env` in your editor and add the AI provider key and source credentials you need for the first source you plan to scout.

## 2. Configure one AI provider first

For a first useful scout, configure at least one AI provider path before expecting scores, analysis, or reports. API keys are the recommended first-run path because they are explicit, easy to verify, and portable across local and Docker runs.

| Provider path | First-run role | Notes |
| --- | --- | --- |
| `ANTHROPIC_API_KEY` | Recommended explicit first provider key | Enables Anthropic-backed AI workflows when the runtime uses that provider path. |
| `GEMINI_API_KEY` | Alternative AI provider key | Enables the Gemini-compatible provider path when configured. |
| Copilot CLI | Supported advanced fallback path | Works only when the Copilot CLI is installed and authenticated in the same runtime environment as ThreadLens. |
| Claude CLI | Supported advanced fallback path | Works only when the Claude CLI is installed and authenticated in the same runtime environment as ThreadLens. |
| Host CLI bridge | Advanced fallback path for Dockerized installs | Routes AI calls from a Docker container to a host-authenticated CLI runtime over a loopback HTTP service. See the bridge guidance below. |

For first-run Docker docs, prefer an explicit provider key because CLI availability and authentication can vary by host and container setup. The CLI-backed paths are supported runtime paths, but this page intentionally does not provide Docker mount, install, or authentication walkthroughs.

### Host CLI bridge (Docker dev fallback)

When ThreadLens runs inside Docker on macOS or Linux and the host machine has Copilot CLI or Claude CLI installed and authenticated, `pnpm run docker:dev` best-effort starts a host helper named `scout-ai-bridge`. The helper lets the Dockerized API route CLI-backed AI calls to the host without mounting CLI credential directories into the container.

The bridge is still a fallback path. API keys remain the recommended explicit first-run choice because they are easier to verify and work the same inside and outside Docker.

**What Docker dev creates:**
- `$XDG_CONFIG_HOME/scout/ai-bridge.token`, or `~/.config/scout/ai-bridge.token`, containing a local bearer token.
- `$XDG_CONFIG_HOME/scout/ai-bridge.json`, or `~/.config/scout/ai-bridge.json`, pointing host-side clients at `http://127.0.0.1:4761`.
- `open-core/.env` bridge values that point the API container at `http://host.docker.internal:4761` and mount the token file read-only at `/run/secrets/scout-ai-bridge-token`.

**Network safety:**
- The helper binds to `127.0.0.1:4761` by default and rejects wildcard or public bind addresses.
- Exposing the bridge publicly is unsafe and unsupported.
- From inside Docker, `localhost` refers to the container itself, so the Docker-facing URL uses `host.docker.internal` instead.

**Opt out:**
- Set `SCOUT_AI_BRIDGE_DISABLE=1` before running `pnpm run docker:dev` to skip bridge bootstrap.
- Bridge bootstrap failures do not block Docker startup. If the helper cannot build, launch, or find an authenticated CLI, ThreadLens falls back to direct in-container providers and configured API keys.

**UI status:**
- The Models UI shows bridge status as a read-only indicator. It is not a separate model mode or model picker.

## 3. Add source-specific credentials only when needed

AI provider configuration is separate from source access. Add source credentials for the source you want to scout:

| Source | Credentials | First-run guidance |
| --- | --- | --- |
| Reddit | No extra source credential is documented for the current open-core first-run path | Lowest-friction first source, but still needs an AI provider for useful scoring and reports. |
| Google Search | `PARALLEL_API_KEY` | Add this when you want Google scouting through the configured search provider. |
| Bluesky | `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` | Add both before relying on Bluesky scouting. |

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
| `THREADLENS_RUNTIME_MODE` | Selects self-hosted or hosted runtime mode; self-hosted is the default for open-core use. |

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
- See [Model and Provider Configuration](../user-guide/model-provider-configuration/) for provider fallback behavior.
- See [AI Providers](../architecture/ai-providers/) for the architecture-level provider order.
- See [Environment Variables](../reference/environment-variables/) for the complete variable reference.
