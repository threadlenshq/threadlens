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

### Host CLI bridge (advanced, Docker only)

When ThreadLens runs inside Docker and the host machine has a supported CLI (Copilot CLI or Claude CLI) installed and authenticated, a host CLI bridge can proxy AI calls from the container to the host CLI runtime over localhost HTTP.

The bridge is an advanced fallback path, not the recommended first-run setup. API keys remain the recommended first-run choice.

**Bridge URL and network safety:**
- The bridge must listen only on a loopback or private-network address (for example `127.0.0.1` or a Docker internal network). Do not expose the bridge on a public interface.
- Exposing the bridge publicly is unsafe and unsupported. The bridge command surface is protected by bearer-token authentication, but public exposure is not a supported configuration.

**Auto-launch behaviour:**
- A helper process can attempt to auto-launch the bridge when needed. This launch is best-effort only: if the helper binary is not present or the host CLI is not authenticated, the bridge will not become available and AI calls will fall back to the next provider in the fallback order.
- Auto-launch should not run as a hidden always-on background process. If you encounter unexpected background processes, check your runtime configuration.

**Docker networking note:**
- From inside a Docker container, `localhost` refers to the container itself, not the host. Use the Docker host gateway address (typically `host-gateway` or `172.17.0.1`) or Docker's `host.docker.internal` alias to reach a bridge running on the host.
- This Docker localhost vs host networking distinction is a common source of confusion. If the bridge status shows unavailable inside Docker, verify the bridge URL points to the host, not the container.

**UI status:**
- When the bridge is configured, the Models UI shows bridge status as a read-only indicator. Bridge status is not a selectable model mode — it is a signal about fallback availability.

**Open questions (not yet resolved):**
- The exact helper binary name and packaging location for auto-launch are implementation-planning open questions. This page will be updated when those details are finalized.

The bridge degrades quietly: if the bridge is unreachable, ThreadLens logs the failure and moves to the next provider in the fallback order rather than returning an error to the user.

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
