---
title: Model and Provider Configuration
description: Understand provider keys and the current AI fallback model used by ThreadLens.
---


ThreadLens uses AI providers for scoring, analysis, query assistance, clustering, and report generation.

## Provider keys

Configure provider keys in `open-core/.env`. Keep the existing runtime variable names such as `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, and `PARALLEL_API_KEY`.

## Fallback behavior

The Go API preserves the existing provider fallback shape: Copilot CLI when available, Claude CLI, Anthropic-compatible HTTP calls, and Gemini-compatible HTTP calls.

## Safe operation

- Start with one provider key when testing locally.
- Use fake values in documentation and bug reports.
- Do not commit `.env` files containing real provider keys.
