---
title: Model and Provider Configuration
description: Understand provider keys and the current AI fallback model used by ThreadLens.
---

ThreadLens uses AI providers for scoring, analysis, query assistance, clustering, and report generation. Configure at least one AI provider path before expecting useful first-run scouting, AI-scored findings, or reports. A ready provider path can be a configured provider key or a supported CLI-backed path available in the same runtime environment as ThreadLens.

## Provider keys

Configure provider keys in `open-core/.env`. Keep the existing runtime variable names such as `ANTHROPIC_API_KEY` and `GEMINI_API_KEY`.

For a first Docker-based run, start with one explicit provider key. `ANTHROPIC_API_KEY` is the recommended first documented path, and `GEMINI_API_KEY` is another supported key-backed path. Copilot CLI and Claude CLI are also supported advanced fallback paths when the CLI is installed and authenticated in the same runtime environment as ThreadLens. A host CLI bridge is an additional optional fallback path available when ThreadLens runs in Docker or a self-hosted runtime and the host has an authenticated CLI; see the bridge guidance below.

## AI providers are separate from source credentials

AI provider keys unlock scoring, analysis, query assistance, clustering, and reports. Source credentials unlock specific sources:

| Capability | Credential |
| --- | --- |
| AI scoring and reports through Anthropic-backed calls | `ANTHROPIC_API_KEY` |
| AI scoring and reports through Gemini-compatible calls | `GEMINI_API_KEY` |
| Google scouting through the configured search provider | `PARALLEL_API_KEY` |
| Bluesky scouting | `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` |

Reddit is the lowest-friction first source in the current open-core docs because it has no extra source credential listed here, but it still needs AI provider readiness for useful scoring and reports.

## Models view and fallback behavior

The Models view is the source of truth for per-task model selection. ThreadLens stores each task setting in `app_settings` as `model.<taskId>` with a catalog model ID such as `copilot:gpt-5-mini`; it does not store bridge state or bridge credentials.

For each task, ThreadLens tries the selected or default catalog model first, then the preserved fallback order below, skipping duplicate model IDs:

1. `copilot:gpt-5-mini`
2. `claude-cli:haiku`
3. `sdk:haiku`
4. `gemini:2.5-flash`

The returned model ID and usage metering remain the catalog model ID that succeeded. If `copilot` or `claude-cli` succeeds through the local bridge, the used model ID is still the catalog model ID, not a bridge model.

## Optional local host CLI bridge

The host CLI bridge is an optional local transport for `copilot` and `claude-cli` catalog models. It lets local Docker development reuse AI CLI sessions authenticated on the host machine without mounting host credential directories into the container.

For most local Docker users, no manual bridge command is required. `pnpm run docker:dev` best-effort bootstraps the bridge automatically and only enables it when the host bridge is healthy and exposes at least one available runtime.

The bridge is not required for production or VPS self-host deployments. Production should use API keys, or CLIs installed and authenticated directly in the API runtime environment. Docker prod does not start bridge bootstrap, does not require a bridge service, and does not mount bridge token files by default. Self-hosters may still run the bridge later if they want host-authenticated CLI reuse.

**Key points:**
- Users select catalog models in the Models view; they do not select bridge as a provider.
- The Models UI may show bridge status as read-only external runtime status.
- Bridge status must not include token values, token file paths, bridge URLs, host usernames, or raw CLI output.
- `SCOUT_AI_BRIDGE_DISABLE=1` disables bridge discovery, health checks, and generation calls.
- `SCOUT_AI_BRIDGE_MODE=local` enables local config-file discovery for desktop and Docker development.
- Explicit `SCOUT_AI_BRIDGE_URL` plus `SCOUT_AI_BRIDGE_TOKEN_FILE` is an advanced local-only override.
- Bridge failures are recoverable transport failures; ThreadLens falls through to direct CLI or API-key providers.

**Advanced manual helper:**
- Run `pnpm run bridge:start` from `open-core/` to start the bridge explicitly.
- Run `pnpm run bridge:status` or `pnpm run bridge:health` to inspect it.
- Run `pnpm run bridge:stop` to stop the managed bridge process.
- See [Local AI Bridge](../reference/local-ai-bridge/) for the full workflow.

## Safe operation

- Start with one provider key when testing locally or following the first-run Docker path.
- Use fake values in documentation and bug reports.
- Do not commit `.env` files containing real provider keys.
- Do not publish private prompts, hosted credentials, billing tokens, or account-specific quota details.

For the complete first-run setup bridge, see [Configuration Basics](../start-here/configuration-basics/). For the complete variable list, see [Environment Variables](../reference/environment-variables/).
