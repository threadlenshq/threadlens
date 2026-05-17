---
title: First Value in 15 Minutes
description: Complete setup, create one project, add one narrow query, run one scout, and inspect first findings.
---

This guide starts after `pnpm run docker:dev` is running from the `open-core/` directory.

## 1. Verify the local stack

Run:

```bash
pnpm run self-host:smoke
```

Expected result: the script finds `pnpm`, `docker`, and `curl`, then reports the API, onboarding status, and web app checks. If a service is not reachable, keep Docker running and retry after the containers finish starting.

## 2. Complete required setup

Open `http://localhost:4748`. On a new database, ThreadLens shows the setup wizard before the workspace.

Choose the fastest reliable AI path for self-hosting:

- Anthropic API key through `ANTHROPIC_API_KEY`, or
- Gemini API key through `GEMINI_API_KEY`.

Local CLI paths are advanced options for users who already have the CLI authenticated inside the runtime. They are not required for the first self-hosted run.

## 3. Create one research project

Use the checklist starter project or create one manually.

Recommended first project:

| Field | Value |
| --- | --- |
| Slug | `ai-note-taking` |
| Display name | `AI Note Taking Research` |
| Mode | `research` |

## 4. Add one narrow query

Start with one Reddit query because it is the lowest-friction first source.

Good first queries:

- `meeting notes too time consuming`
- `forget action items after calls`
- `transcribing interviews workflow`

Avoid broad first queries such as `AI tools`, `productivity`, or `startup ideas` because they produce noisy results.

## 5. Run one scout manually

Use the Scout button for the source you configured. ThreadLens does not run external scouts automatically during onboarding.

Wait for the run to complete before editing the query. If the run fails, read the error and check provider keys before changing project settings.

## 6. Inspect first findings

Open the strongest findings first:

- look for concrete complaints in the user's words
- star promising findings
- exclude obvious noise
- delay reports until several findings share a repeated pain theme

## 7. Expand only after first review

After one completed scout, expand toward 8 enabled queries across at least 3 angles. Add Google only after `PARALLEL_API_KEY` is configured and Bluesky only after `BLUESKY_HANDLE` plus `BLUESKY_APP_PASSWORD` are configured.
