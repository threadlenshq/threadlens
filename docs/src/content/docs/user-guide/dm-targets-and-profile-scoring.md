---
title: DM Targets and Profile Scoring
description: Understand how ThreadLens generates outreach targets and scores Reddit author profiles.
---

DM targets are generated for projects in **marketing mode** to identify users worth reaching out to. Targets are scored on intent, then adjusted by the author's Reddit profile quality.

## How targets are built

For each scored post (final score ≥ 5), ThreadLens builds a list of candidates:

1. **Post author** — always included as the primary candidate.
2. **Top commenters** — up to 5 top-level comments are fetched from the post thread. Each comment author becomes a candidate.

## Candidate filtering

Before scoring, deterministic rules filter out poor candidates. A candidate is **excluded** for:

- Empty or [deleted]/[removed] usernames
- Bot-like usernames (e.g. ending in `_bot`, containing `automoderator`)
- Self-identified automation (`"i am a bot"`, `"mirror bot"`, etc.)
- Promotional spam patterns

A candidate is **penalized** (but not excluded) for:

- Very new account (≤ 48 hours) without intent language: penalty **2.0**
- Promotional signals in display name or bio: penalty **1.0**
- Boilerplate link-heavy or tag-heavy text: penalty **1.0**

## Intent scoring

Each candidate starts from the post's final score. Commenters start 1 point lower than the author. Intent bonuses are then added:

| Signal | Bonus |
|---|---|
| Strong intent ("i need", "i'm struggling", "i can't") | +0.5 |
| Seeking help ("can someone", "recommend", "looking for") | +0.75 |
| Tool/workflow mentions ("tool", "app", "software", "workflow") | +0.5 |
| Frustration language ("frustrated", "pain", "broken") | +0.5 |
| Engagement (post score or likes) | up to +0.5 |

Filter penalties are subtracted, then the score is clamped to 1–10. The top 3 candidates per post are stored.

## Profile enrichment (Reddit)

For Reddit candidates, ThreadLens fetches the author's profile to adjust the intent score. Two endpoints are queried:

| Endpoint | Data extracted |
|---|---|
| `about.json` | Account age, comment karma, post karma, verified email, gold status, NSFW flag |
| `submitted.json?limit=25` | Last 25 posts — titles, subreddits, and domains for self-promotion detection |

Profile data is cached per-run, so each unique author is fetched only once even if they appear across multiple posts.

### Self-promotion detection

A submitted post is flagged as self-promotion if:

- **Title contains**: "check out my", "my plugin", "my tool", "i made a", "i built a", "launched my", "shipped my", and similar
- **Domain matches**: youtube.com, patreon.com, gumroad.com, producthunt.com, substack.com, twitter.com, and similar

Self-promo ratio = flagged posts ÷ last 25 submitted posts.

### Profile scoring rules

Profile score is deterministic (not AI-based) and clamped to **-5 to +2**.

| Condition | Penalty/Bonus |
|---|---|
| Account age < 30 days | **-2** |
| Account age < 90 days | **-1** |
| Comment karma negative | **-2** |
| Post + comment karma both zero | **-1** |
| Self-promo ratio > 50% | **-3** |
| Self-promo ratio > 25% | **-1** |
| Single subreddit + any self-promo | **-1** |
| Verified email | **+1** |
| Gold status | **+1** |

### Effect on DM intent

The profile score multiplicatively adjusts the intent score:

```
adjusted = clamp( (10 + profile_score) / 10 × intent_score,  1, 10 )
```

| Profile score | Effect |
|---|---|
| **+2** (best) | Intent boosted by 20% |
| **0** | No change |
| **-5** (worst) | Intent cut by 50% |

## Where to find DM targets

DM targets appear in the project detail view for marketing mode projects. Each target shows the username, adjusted intent score, derived context, and suggested approach. Profile signals are stored as structured metadata.

## Limitations

- Profile enrichment applies to Reddit candidates only. Bluesky candidates are scored on intent alone.
- Profile data is cached only for the duration of a single scout run. Each run refetches profiles independently.
