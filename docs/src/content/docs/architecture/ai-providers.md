---
title: AI Providers
description: Understand how ThreadLens chooses AI providers for scoring and analysis.
---

ThreadLens uses AI providers for scoring, report generation, query assistance, clustering, and analysis.

## Provider order

The Go API preserves the current fallback shape used by the project:

1. Copilot CLI when available and authenticated.
2. Claude CLI when available and authenticated.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

## First-run note

The Start Here docs recommend configuring an explicit provider key for first use because CLI-backed provider paths depend on local or container availability and authentication. A no-key Docker startup can verify that the app loads, but useful scoring, analysis, and reports need an available AI provider path.

## Provider keys versus source credentials

AI provider keys are separate from source-specific credentials. `ANTHROPIC_API_KEY` and `GEMINI_API_KEY` unlock AI workflows, while `PARALLEL_API_KEY` unlocks Google scouting through the configured search provider and `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` unlock Bluesky scouting.

## Operational guidance

Keep provider examples fake in public docs. Do not publish real provider keys, private prompts, hosted credentials, billing tokens, customer data, or account-specific quota details.

For user-facing setup guidance, see [Configuration Basics](../start-here/configuration-basics/) and [Model and Provider Configuration](../user-guide/model-provider-configuration/).
