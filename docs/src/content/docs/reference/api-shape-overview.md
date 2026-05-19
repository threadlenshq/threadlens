---
title: API Shape Overview
description: Understand the high-level Go API surface without generated API reference.
---


The active backend is the Go API in `apps/api`. Detailed generated API reference is deferred for the first docs release.

## Main resource areas

- Projects for research workspaces.
- Queries for source-specific project searches.
- Prompts for project-specific AI prompt configuration.
- Scout runs for active and historical scouting work.
- Posts and Google results for collected findings.
- Reports for social research and Google search analysis.
- Schedules for recurring scouting.
- Runtime and model endpoints for configuration-aware UI behavior.

## Compatibility goal

The Go API preserves Express response shapes where the frontend already depends on them. Handlers stay thin, services own orchestration, and repositories own SQLite access.
