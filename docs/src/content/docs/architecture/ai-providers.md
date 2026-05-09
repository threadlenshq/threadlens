---
title: AI Providers
description: Understand how ThreadLens chooses AI providers for scoring and analysis.
---

ThreadLens uses AI providers for scoring, report generation, query assistance, clustering, and analysis. The provider policy is hybrid: API keys are the recommended first-run path because they work in any environment without additional tooling, while Copilot CLI and Claude CLI are supported advanced fallback paths when those tools are available and authenticated in the runtime environment.

## Provider order

The Go API resolves providers in the following order. API-key-backed paths are always available when the key is set. CLI-backed paths require the corresponding CLI tool to be installed, on PATH, and authenticated in the runtime environment:

1. Copilot CLI when available and authenticated.
2. Claude CLI when available and authenticated.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

## First-run note

Configure at least one API key before first use. Copilot CLI and Claude CLI are powerful fallback paths but depend on local or container availability — they are not present in a default Docker image or a fresh CI environment. Without an available provider path, scoring, analysis, and report generation will not function.

## Provider keys versus source credentials

AI provider keys are separate from source-specific credentials. `ANTHROPIC_API_KEY` and `GEMINI_API_KEY` unlock AI workflows, while `PARALLEL_API_KEY` unlocks Google scouting through the configured search provider and `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` unlock Bluesky scouting.

## Operational guidance

Keep provider examples fake in public docs. Do not publish real provider keys, private prompts, hosted credentials, billing tokens, customer data, or account-specific quota details.

For user-facing setup guidance, see [Configuration Basics](../start-here/configuration-basics/) and [Model and Provider Configuration](../user-guide/model-provider-configuration/).
