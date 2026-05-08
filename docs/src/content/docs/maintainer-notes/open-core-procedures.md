---
title: Open-Core Procedures
description: Public maintainer notes for keeping open-core documentation and support healthy.
---


This section is public and maintainer-oriented. It is not a place for private hosted operations.

## Docs hygiene

- Keep README files concise and link to canonical docs pages for details.
- Run `pnpm run docs:check` and `pnpm run docs:build` from `open-core/` before publishing docs changes.
- Search for unsafe placeholders, real secrets, and private hosted details before release.

## Open-core support triage

- Ask for reproduction commands and redacted logs.
- Confirm whether Docker or local commands were used.
- Keep support answers focused on the self-hosted open-core app unless hosted behavior is public and released.
