---
title: Self-Hosted Procedures
description: Public maintainer notes for keeping self-hosted documentation and support healthy.
---


This section is public and maintainer-oriented. It is not a place for private hosted operations.

## Docs hygiene

- Keep README files concise and link to canonical docs pages for details.
- Run `pnpm run docs:check` and `pnpm run docs:build` from `open-core/` before publishing docs changes.
- Keep `open-core/LICENSE`, `reference/licensing`, and README license summaries consistent when the license or contribution policy changes.
- Search for unsafe placeholders, real secrets, and private hosted details before release.

## Self-Hosted Support Triage

- Ask for reproduction commands and redacted logs.
- Confirm whether Docker or local commands were used.
- Keep support answers focused on the self-hosted app unless hosted behavior is public and released.
