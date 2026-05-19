---
title: Scouting Sources
description: Understand what each scouting source is best at and which credentials each source needs.
---

ThreadLens can scout social posts and Google Search results from project-specific queries. Useful scoring and reports require AI provider readiness even when a source itself has little or no extra credential setup.

## Source readiness at a glance

| Source | Best for | Source credentials | First-run guidance |
| --- | --- | --- | --- |
| Reddit | Long-form complaints, niche community language, and detailed workarounds | No extra source credential is documented for the current first-run path | Start here when you want the least source-specific setup. Configure an AI provider first for useful scoring and reports. |
| Google Search | Pages, comparisons, search intent, commercial intent, and recurring problem language outside social posts | `PARALLEL_API_KEY` | Use after the configured search provider key is present in `open-core/.env`. |
| Bluesky | Shorter public posts, emerging conversation, and social commentary | `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` | Use after both Bluesky credentials are present in `open-core/.env`. |

## Reddit

Reddit is useful for long-form complaints, niche community language, and detailed workarounds. It is the recommended first source when you want the least source-specific credential overhead.

Reddit results still need AI provider readiness before you should expect useful scoring, analysis, or reports.

## Bluesky

Bluesky is useful for shorter public posts and emerging conversation. Configure `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` before relying on Bluesky scouting.

## Google Search

Google scouting finds pages and search results that can show comparison intent, commercial intent, and recurring problem language. Configure `PARALLEL_API_KEY` when using the Parallel.ai Search provider.

## Source selection

Start with one source, inspect the results, then add more sources after the project query language is producing useful findings. If a source returns errors or empty results, check both the AI provider credential and the source-specific credential for that source.

For first-run credential setup, see [Configuration Basics](../start-here/configuration-basics/). For the complete variable reference, see [Environment Variables](../reference/environment-variables/).
