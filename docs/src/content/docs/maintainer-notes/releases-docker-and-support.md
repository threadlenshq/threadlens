---
title: Releases, Docker, and Support
description: Public maintainer notes for docs releases, Docker image hygiene, and support routing.
---


Use these public notes to keep self-hosted releases and docs aligned.

## Docs release checks

Run from the repository root:

```bash
pnpm run docs:check
pnpm run docs:build
```

## Docker release checks

Run from the repository root:

```bash
pnpm run docker:dev
pnpm run docker:down
pnpm run docker:prod
pnpm run docker:down
```

## Support routing

Public self-hosted issues belong in public issue tracking. Hosted SaaS billing, credentials, and private infrastructure support do not belong in public docs or public issue comments.
