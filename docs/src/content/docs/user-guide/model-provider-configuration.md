---
title: Model and Provider Configuration
description: Understand provider keys and the current AI fallback model used by ThreadLens.
---

ThreadLens uses AI providers for scoring, analysis, query assistance, clustering, and report generation. Configure at least one AI provider path before expecting useful first-run scouting, AI-scored findings, or reports. A ready provider path can be a configured provider key or a supported CLI-backed path available in the same runtime environment as ThreadLens.

## Provider keys

Configure provider keys in `open-core/.env`. Keep the existing runtime variable names such as `ANTHROPIC_API_KEY` and `GEMINI_API_KEY`.

For a first Docker-based run, start with one explicit provider key. `ANTHROPIC_API_KEY` is the recommended first documented path, and `GEMINI_API_KEY` is another supported key-backed path. Copilot CLI and Claude CLI are also supported advanced fallback paths when the CLI is installed and authenticated in the same runtime environment as ThreadLens. A host CLI bridge is an additional advanced fallback path available when ThreadLens runs inside Docker and the host has an authenticated CLI; see the bridge guidance below.

## AI providers are separate from source credentials

AI provider keys unlock scoring, analysis, query assistance, clustering, and reports. Source credentials unlock specific sources:

| Capability | Credential |
| --- | --- |
| AI scoring and reports through Anthropic-backed calls | `ANTHROPIC_API_KEY` |
| AI scoring and reports through Gemini-compatible calls | `GEMINI_API_KEY` |
| Google scouting through the configured search provider | `PARALLEL_API_KEY` |
| Bluesky scouting | `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` |

Reddit is the lowest-friction first source in the current open-core docs because it has no extra source credential listed here, but it still needs AI provider readiness for useful scoring and reports.

## Fallback behavior

The Go API uses this provider fallback order:

1. Copilot CLI when available and authenticated.
2. Claude CLI when available and authenticated.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

Because CLI-backed paths depend on local or container runtime availability and authentication, first-run docs recommend configuring an explicit provider key instead of assuming a CLI path is available. Copilot CLI and Claude CLI are the currently documented supported CLI-backed paths; other AI CLIs are not documented as current supported providers.

## Host CLI bridge (advanced, Docker only)

When ThreadLens runs inside Docker, a host CLI bridge can proxy AI calls from the container to a host-authenticated CLI runtime over a loopback HTTP service. The bridge extends the CLI-backed fallback paths to Dockerized installs without requiring the CLI to be installed inside the container.

The bridge is an advanced fallback path. API keys remain the recommended first-run choice.

**Key points:**
- The bridge must listen only on a loopback or private-network address. Do not expose it on a public interface — this is unsafe and unsupported.
- Bearer-token authentication protects the bridge command surface, but public exposure is not a supported configuration.
- Auto-launch of the helper process is best-effort only. If the binary is absent or the host CLI is not authenticated, the bridge will be unavailable and ThreadLens will fall back to the next provider.
- Auto-launch should not run as a hidden always-on background process.
- From inside Docker, `localhost` refers to the container, not the host. Use the Docker host gateway (e.g., `host.docker.internal` or `172.17.0.1`) to reach a bridge on the host. A mismatch here is a common source of "bridge unavailable" status.
- The Models UI shows bridge status as a read-only indicator, not as a selectable model mode.
- Bridge outages degrade quietly: ThreadLens logs the failure and moves to the next provider rather than surfacing an error to the user.
- The exact helper binary name and packaging location are not yet finalized and will be documented when resolved.

## Safe operation

- Start with one provider key when testing locally or following the first-run Docker path.
- Use fake values in documentation and bug reports.
- Do not commit `.env` files containing real provider keys.
- Do not publish private prompts, hosted credentials, billing tokens, or account-specific quota details.

For the complete first-run setup bridge, see [Configuration Basics](../start-here/configuration-basics/). For the complete variable list, see [Environment Variables](../reference/environment-variables/).
