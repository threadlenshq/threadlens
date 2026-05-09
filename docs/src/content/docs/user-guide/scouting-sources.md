---
title: Scouting Sources
description: Understand what each scouting source is best at, which credentials each source needs, and what is required for useful scoring and reports.
---

ThreadLens can scout social posts and Google Search results from project-specific queries. Each source has its own credential requirements, and useful AI scoring and reports require an AI provider to be configured regardless of which source you use.

## Source readiness at a glance

| Source | Runs without credentials | Useful results require |
| --- | --- | --- |
| Reddit | ✅ Yes | An AI provider key for scoring |
| Bluesky | ❌ No | `BLUESKY_HANDLE` + `BLUESKY_APP_PASSWORD` + an AI provider key |
| Google Search | ❌ No | `PARALLEL_API_KEY` + an AI provider key |

See [Configuration Basics](/start-here/configuration-basics/) for step-by-step setup and [Environment Variables](/reference/environment-variables/) for a full variable reference.

## Reddit

Reddit is useful for long-form complaints, niche community language, and detailed workarounds.

**Credential requirements:** None. Reddit scouting uses the public API and does not require a Reddit account or token.

**Ready to use:** Reddit queries run as soon as the server starts. Without an AI provider key, posts are collected but scores and reports will not be generated.

## Bluesky

Bluesky is useful for shorter public posts and emerging conversation.

**Credential requirements:** Set `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` in `open-core/.env` before running a Bluesky scout. Without these, Bluesky runs will fail immediately.

**Ready to use:** After adding the credentials and restarting, Bluesky queries become active. As with all sources, an AI provider key is required for scoring and reports.

## Google Search

Google scouting finds pages and search results that surface comparison intent, commercial intent, and recurring problem language.

**Credential requirements:** Set `PARALLEL_API_KEY` in `open-core/.env` to use the default Parallel.ai Search provider. Without this key, Google scout runs will return no results.

**Ready to use:** After adding the key and restarting, Google queries become active. An AI provider key is required for the per-result analysis and the executive summary report.

## AI provider readiness

Scoring and reports are driven by the configured AI provider. Without a working AI provider key, ThreadLens collects raw posts but cannot:

- Calculate pain-point scores
- Generate clustering or research reports
- Produce Google Search executive summaries

Configure at least one AI provider (Anthropic, Gemini, or Copilot CLI) in `open-core/.env` before expecting scored results. See [Configuration Basics](/start-here/configuration-basics/) for instructions.

## Source selection

Start with one source, inspect the results, then add more sources after the project query language is producing useful findings.
