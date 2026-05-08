---
title: AI Providers and Fallbacks
description: Understand how ThreadLens chooses AI providers for scoring and analysis.
---

# AI Providers and Fallbacks

ThreadLens uses AI providers for scoring, report generation, query assistance, and analysis.

## Provider order

The Go API preserves the current fallback shape used by the project:

1. Copilot CLI when available.
2. Claude CLI when available.
3. Anthropic SDK-compatible HTTP calls when `ANTHROPIC_API_KEY` is configured.
4. Gemini-compatible HTTP calls when `GEMINI_API_KEY` is configured.

## Operational guidance

Keep provider examples fake in public docs. Do not publish real provider keys, private prompts, hosted credentials, or account-specific quota details.
