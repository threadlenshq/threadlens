---
title: Scoring, Filtering, and Statuses
description: Understand how ThreadLens turns raw findings into reviewable research signals.
---


ThreadLens scores and filters findings so a project does not become a pile of raw links.

## Scoring

Posts are scored for signals such as pain, relevance, frustration, solution seeking, and workaround language. Google results are analyzed for relevance, intent, and opportunity.

## Filtering

Filtering removes low-signal or promotional results before they consume review time. Deduplication keeps repeated URLs and posts from appearing as separate findings when they represent the same signal.

## Statuses

Use statuses to track review progress:

| Status | Meaning |
| --- | --- |
| `new` | Found but not reviewed. |
| `starred` | High-signal and worth returning to. |
| `excluded` | Not useful for the current project. |
| `drafted` | Used for a draft response or follow-up. |
| `commented` | Already acted on. |

Keep statuses factual so reports and future reviews stay useful.

## Filtering is visibility, not scoring

Scout can hide spam, bot-like, low-quality-account, and likely AI-generated findings from default review queues. This filtering does not change `post_score`, `final_score`, Google relevance scores, draft fields, or review statuses.

Filtered findings are retained for the self-host owner under **Filtered findings**. Use **Restore visibility** to recover only one item, or **Restore and trust…** to recover the item and create a project-scoped trust record for the displayed source or exact system-generated filter signature. Trust records are allowlist overrides; they do not boost scores.

Use **Re-filter selected** from the normal post list to check selected visible posts again. Use **Re-check selected** from **Filtered findings** to apply new trust records to already-filtered rows.
