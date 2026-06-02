---
title: Pipelines
description: Understand the social and Google scouting flows at a high level.
---


ThreadLens has social scouting pipelines and a Google scouting pipeline.

## Social scouting

1. A user or schedule starts a scout run for a project and platform.
2. The API loads enabled project queries for that source.
3. The pipeline fetches candidate posts.
4. Scoring calculates pain, relevance, and opportunity signals.
5. Filtering and deduplication remove noise.
6. Posts and run status are saved for review.

Before visible persistence, candidates pass through the shared conservative filtering subsystem. The classifier normalizes platform source identity, checks project trust records, applies deterministic spam/bot/low-quality/AI-boilerplate rules, and fails open to visible when a decision is ambiguous or an AI filter hook is unavailable. Filtered rows remain in `posts` or `google_results` with reason metadata and are excluded from default review/report views.

## Google scouting

1. Root keywords expand into search variations.
2. The search provider returns result URLs.
3. Page content is fetched when possible.
4. Results are analyzed for relevance, intent, and opportunity.
5. Canonical URL deduplication groups repeated results.
6. A Google report summarizes findings, risks, opportunities, and next actions.

Before visible persistence, candidates pass through the shared conservative filtering subsystem. The classifier normalizes platform source identity, checks project trust records, applies deterministic spam/bot/low-quality/AI-boilerplate rules, and fails open to visible when a decision is ambiguous or an AI filter hook is unavailable. Filtered rows remain in `posts` or `google_results` with reason metadata and are excluded from default review/report views.
