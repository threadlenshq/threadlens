---
title: AI Providers
description: Understand how ThreadLens chooses AI providers for scoring and analysis.
---

ThreadLens uses a hybrid AI runtime. The user-visible model catalog contains catalog providers such as `copilot`, `claude-cli`, `sdk`, and `gemini`; the runtime chooses a transport for the selected catalog model.

## Runtime layers

The core provider runtime runs inside the Go API process environment:

1. Copilot CLI when installed and authenticated in that environment.
2. Claude CLI when installed and authenticated in that environment.
3. Anthropic SDK calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini SDK calls when `GEMINI_API_KEY` is configured.

The optional external runtime is the local host CLI bridge. It can satisfy `copilot` and `claude-cli` catalog model requests before direct CLI is attempted, but only when bridge policy and config allow it. `sdk` and `gemini` models never route through the bridge. The bridge is optional for self-hosted deployments as well as local Docker development.

## Fallback order

Task-aware calls use the model selected in the Models view first, then this exact fallback order with duplicates skipped:

1. `copilot:gpt-5-mini`
2. `claude-cli:haiku`
3. `sdk:haiku`
4. `gemini:2.5-flash`

## Production and local development

Production and VPS self-host deployments do not need a bridge daemon, bridge token mount, or bridge bootstrap. They should use API keys or install and authenticate CLIs directly inside the runtime environment when CLI-backed models are desired. If an operator wants host-authenticated CLI reuse later, they can still add the bridge as an optional local transport.

Local desktop and local Docker development may use `SCOUT_AI_BRIDGE_MODE=local` and best-effort Docker dev bootstrap to reuse host CLI sessions. The bridge is local-only convenience infrastructure, not a managed AI route and not a separate model provider.

## Provider keys versus source credentials

AI provider keys are separate from source-specific credentials. `ANTHROPIC_API_KEY` and `GEMINI_API_KEY` unlock AI workflows, while `PARALLEL_API_KEY` unlocks Google scouting through the configured search provider and `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` unlock Bluesky scouting.

## Operational guidance

Keep provider examples fake in public docs. Do not publish real provider keys, private prompts, hosted credentials, billing tokens, customer data, or account-specific quota details.

For user-facing setup guidance, see [Configuration Basics](../start-here/configuration-basics/) and [Model and Provider Configuration](../user-guide/model-provider-configuration/).
