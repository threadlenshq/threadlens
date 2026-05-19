---
title: Docs Contributions
description: Keep ThreadLens docs readable in GitHub and safe to publish.
---


Docs source lives in `docs/src/content/docs/` and is publishable by default.

Public documentation changes should be submitted from a fork through a pull request. Direct write access is not part of the public contribution path.

## Authoring rules

- Use Markdown for normal prose pages.
- Start each page with frontmatter, a clear H1, and a one-paragraph purpose statement.
- Prefer task-oriented titles such as "Run ThreadLens with Docker".
- Include prerequisites, commands, expected URLs, and verification steps on setup pages.
- Use relative links when they improve GitHub readability.
- Keep commands accurate to the current repository workspace.
- Use fake sample tokens only.

## Do not include

- Real provider keys.
- Private hosted credentials.
- Billing operations.
- Private customer data.
- Private infrastructure runbooks.
- Unpublished commercial roadmap details.

## Contribution license note

If you submit a docs pull request, you represent that you have the right to contribute it and that the contribution may be used, relicensed, and distributed by the licensor as part of ThreadLens.
