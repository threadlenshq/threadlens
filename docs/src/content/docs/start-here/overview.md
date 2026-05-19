---
title: Overview
description: Understand what ThreadLens does, who it is for, and what the current self-hosted release includes today.
---

ThreadLens helps builders find product opportunities by collecting public conversations and search results, scoring them for pain and relevance, filtering duplicates, and turning high-signal findings into product-angle reports.

## Who it is for

ThreadLens is useful when you want to research a niche, validate repeated complaints, watch for recurring workarounds, or understand how people describe problems before you build or market a product.

## What you can do in your first session

In a first useful self-hosted session, you can:

1. Start ThreadLens locally with Docker.
2. Configure one AI provider path so scoring, analysis, and reports can run.
3. Create a research-mode project for one niche or product idea.
4. Add one narrow query for a source such as Reddit.
5. Run a scout and wait for it to complete.
6. Review scored findings, statuses, filters, and post detail.
7. Generate a report only after you have enough high-signal findings to summarize.

If you start without provider keys, use that first launch only to confirm that the web app loads and the API is reachable. Real scouting outcomes need provider configuration before you should expect useful AI-scored results.

## What it scouts

- Reddit posts from project-specific queries.
- Bluesky posts from project-specific queries when Bluesky credentials are configured.
- Google Search results through the configured search provider.

## What it produces

- Scored posts and search results.
- Deduplicated findings with statuses for review.
- Research reports that cluster pain themes and suggest product angles.
- Google reports with summaries, opportunities, risks, and next actions.
- Scheduled scout runs for recurring research workflows.

## Self-Hosted Status

The self-hosted app can run today with Docker or local workspace commands. Hosted ThreadLens is planned as a later managed option for teams that do not want to self-host or manage provider keys.

ThreadLens is source-available under BUSL 1.1. That means you can inspect, modify, fork, and self-host it for your own internal business or personal use, while competing resale and hosted clone use stay restricted. See the [Licensing](../reference/licensing/) page for the current terms.

For architecture details after your first run, see [Workspace Layout](../architecture/workspace-layout/), [Go API](../architecture/go-api/), and [Pipelines](../architecture/pipelines/).
