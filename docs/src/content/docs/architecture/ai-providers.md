---
title: AI Providers
description: Understand how ThreadLens chooses AI providers for scoring and analysis.
---

ThreadLens uses AI providers for scoring, report generation, query assistance, clustering, and analysis. The provider policy is hybrid: API keys are the recommended first-run path, and Copilot CLI plus Claude CLI are supported advanced fallback paths when available in the runtime environment. A host CLI bridge is an additional advanced fallback path for Dockerized installs.

## Provider order

The Go API uses this provider fallback order:

1. Copilot CLI when available and authenticated.
2. Claude CLI when available and authenticated.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

## First-run note

The Start Here docs recommend configuring an explicit provider key for first use because CLI-backed provider paths depend on local or container runtime availability and authentication. A no-key Docker startup can verify that the app loads, but useful scoring, analysis, and reports need an available AI provider path. Copilot CLI and Claude CLI are supported advanced fallback paths when they are installed and authenticated where ThreadLens runs; this architecture page intentionally avoids Docker mount, install, or authentication walkthroughs.

## Host CLI bridge

When ThreadLens runs inside Docker, the host CLI bridge extends the CLI-backed fallback paths by proxying AI calls from the container to a host-authenticated CLI runtime over a loopback HTTP service. The bridge is a fallback path within the provider hierarchy — it is not a separate model mode.

**Architecture notes:**

- The bridge is an advanced fallback path, not the recommended first-run setup. API keys are the recommended first-run choice.
- The bridge must listen on a loopback or private-network address only. Exposing the bridge on a public interface is unsafe and unsupported.
- Bearer-token authentication protects the bridge command surface. Public exposure is not a supported configuration even with bearer tokens.
- Auto-launch of the bridge helper is best-effort only. If the helper binary is absent or the host CLI is not authenticated, the bridge will be unavailable and the provider hierarchy continues to the next entry.
- Auto-launch should not become a hidden always-on background process.
- Docker localhost vs host networking is a common ambiguity. Inside a Docker container, `localhost` refers to the container, not the host. A bridge running on the host must be reachable via the Docker host gateway address (e.g., `host.docker.internal` or `172.17.0.1`).
- Bridge status is exposed as a read-only indicator in the Models UI. Users may interpret bridge status as a selectable model mode — it is not. It is a signal about fallback availability.
- Bridge outages degrade quietly: ThreadLens logs the failure and continues the provider fallback order rather than surfacing an error.
- The helper binary is named `scout-ai-bridge` and lives in the open-core Go API module at `open-core/apps/api/cmd/scout-ai-bridge`.

## Provider keys versus source credentials

AI provider keys are separate from source-specific credentials. `ANTHROPIC_API_KEY` and `GEMINI_API_KEY` unlock AI workflows, while `PARALLEL_API_KEY` unlocks Google scouting through the configured search provider and `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` unlock Bluesky scouting.

## Operational guidance

Keep provider examples fake in public docs. Do not publish real provider keys, private prompts, hosted credentials, billing tokens, customer data, or account-specific quota details.

For user-facing setup guidance, see [Configuration Basics](../start-here/configuration-basics/) and [Model and Provider Configuration](../user-guide/model-provider-configuration/).
