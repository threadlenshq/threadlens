---
title: Scouting Sources
description: Understand what each scouting source is best at and which credentials each source needs.
---


ThreadLens can scout social posts and Google Search results from project-specific queries.

## Reddit

Reddit is useful for long-form complaints, niche community language, and detailed workarounds. It does not require the Bluesky credentials listed below.

## Bluesky

Bluesky is useful for shorter public posts and emerging conversation. Configure `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` before relying on Bluesky scouting.

## Google Search

Google scouting finds pages and search results that can show comparison intent, commercial intent, and recurring problem language. Configure `PARALLEL_API_KEY` when using the Parallel.ai Search provider.

## Source selection

Start with one source, inspect the results, then add more sources after the project query language is producing useful findings.
