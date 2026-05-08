---
title: Create Your First Project
description: Create a project, add a narrow query, run a first scout, review findings, and decide whether to generate a report.
---

Use this walkthrough after Docker is running and [Configuration Basics](configuration-basics/) is complete for at least one AI provider path. Without provider setup, you can still open the app, but real scouting, scoring, analysis, and reports will not produce useful first-run results.

## 1. Open the local web app

Use the URL for the Docker profile you started:

| Docker profile | Web app URL | When to use it |
| --- | --- | --- |
| Development profile, `pnpm run docker:dev` | `http://localhost:4748` | Recommended for the first Start Here walkthrough. |
| Production profile, `pnpm run docker:prod` | `http://localhost:4749` | Use after you have already verified the development profile or want the self-host production shape. |

## 2. Create a project

In the web app, open the project selector in the top-left corner and click **+ New Project**. A creation form will appear.

Fill in the three required fields and click **Create**:

| Field | What to enter | Why it matters |
| --- | --- | --- |
| Slug | A short, lowercase, hyphenated ID such as `ai-note-taking` | This is the machine-readable identifier used in routes and API calls. Keep it simple and stable — you will not need to change it often. |
| Display name | A human-friendly label such as `AI Note Taking Research` | This is what you see while browsing the app. It can be more descriptive than the slug and is easy to rename later. |
| Mode | Choose `research` for the first project | Research mode focuses on discovering pain points, repeated complaints, and product opportunities. Choose `marketing` only when your goal is outreach around an already-understood opportunity. |

Keep the first project narrow. `ai-note-taking` is easier to evaluate than `productivity` because the results will use more specific language.

## 3. Add one first query

Start with one source and one narrow query so you can inspect the results manually.

- Prefer Reddit for the first scout when you want the least source-specific credential overhead.
- Use Google Search only after `PARALLEL_API_KEY` is configured.
- Use Bluesky only after `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` are configured.

Good first queries describe a problem, audience, or workflow. For an AI note-taking project, start with a phrase such as `meeting notes too time consuming`, `forget action items after calls`, or `transcribing interviews workflow` instead of a broad phrase such as `AI tools`.

## 4. Run a scout

Run the scout for the source you configured and wait for completion before changing the query.

A completed first scout should show that the run finished and that ThreadLens checked the source for matching posts or results. If a run fails, check whether the missing credential is an AI provider key, a Google source key, or Bluesky credentials before changing project settings.

## 5. Review findings

Inspect results before generating a report:

- Use score to find posts or results with stronger pain, urgency, or relevance signals.
- Use status to separate new, starred, excluded, drafted, commented, or already-handled findings.
- Use filters to narrow the list when the source returns too much noise.
- Open post detail to read the original wording, not just the score.

Star or keep findings that describe a real problem in the user's own words. Exclude obvious noise before report generation.

## 6. Generate a report when the findings are coherent

Generate a research report after you have selected enough findings about the same market question to summarize. Reports are most useful when the selected findings share a repeated pain theme, workaround, buying trigger, or audience segment.

If the selected findings feel unrelated, refine the query and run another scout instead of generating a report immediately.

## 7. Defer schedules until query quality is proven

Schedules are for recurring research after a query reliably produces useful findings. Do not schedule the first query until you have reviewed at least one completed scout and know the source, query, and provider setup are producing signal instead of noise.

## Related guides

- [Scouting Sources](../user-guide/scouting-sources/) explains when to use Reddit, Google Search, and Bluesky.
- [Reports](../user-guide/reports/) explains research reports and Google reports.
- [Scoring, Filtering, and Statuses](../user-guide/scoring-filtering-and-statuses/) explains the review tools used after a scout.
- [Schedules](../user-guide/schedules/) explains recurring scout runs after query quality is proven.
