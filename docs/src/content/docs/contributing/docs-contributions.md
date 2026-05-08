---
title: Documentation Contribution Rules
description: Keep ThreadLens docs readable in GitHub and safe to publish.
---

# Documentation Contribution Rules

Docs source lives in `open-core/docs/src/content/docs/` and is publishable by default.

## Authoring rules

- Use Markdown for normal prose pages.
- Start each page with frontmatter, a clear H1, and a one-paragraph purpose statement.
- Prefer task-oriented titles such as "Run ThreadLens with Docker".
- Include prerequisites, commands, expected URLs, and verification steps on setup pages.
- Use relative links when they improve GitHub readability.
- Keep commands accurate to the current open-core workspace.
- Use fake sample tokens only.

## Do not include

- Real provider keys.
- Private hosted credentials.
- Billing operations.
- Private customer data.
- Private infrastructure runbooks.
- Unpublished commercial roadmap details.
