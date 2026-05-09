---
title: Model and Provider Configuration
description: Understand provider keys and the current AI fallback model used by ThreadLens.
---

ThreadLens uses AI providers for scoring, analysis, query assistance, clustering, and report generation. Configure at least one AI provider path before expecting useful first-run scouting, AI-scored findings, or reports.

## Provider keys

Configure provider keys in `open-core/.env`. Keep the existing runtime variable names such as `ANTHROPIC_API_KEY` and `GEMINI_API_KEY`.

For a first Docker-based run, start with one explicit provider key. `ANTHROPIC_API_KEY` is the recommended first documented path, and `GEMINI_API_KEY` is another supported key-backed path. CLI-backed paths can work only when the required CLI is available and authenticated in the runtime environment.

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

The Go API preserves the existing provider fallback shape used by the project:

1. Copilot CLI when available and authenticated.
2. Claude CLI when available and authenticated.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

Because CLI-backed paths depend on local or container availability and authentication, first-run docs recommend configuring an explicit provider key instead of assuming a CLI path is available.

## Safe operation

- Start with one provider key when testing locally.
- Use fake values in documentation and bug reports.
- Do not commit `.env` files containing real provider keys.
- Do not publish private prompts, hosted credentials, billing tokens, or account-specific quota details.

For the complete first-run setup bridge, see [Configuration Basics](../start-here/configuration-basics/). For the complete variable list, see [Environment Variables](../reference/environment-variables/).
