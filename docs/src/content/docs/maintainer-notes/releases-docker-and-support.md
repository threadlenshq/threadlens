---
title: Releases, Docker, and Support
description: Public maintainer notes for docs releases, Docker image hygiene, and support routing.
---


Use these public notes to keep self-hosted releases and docs aligned.

## GitHub release checks

Before publishing a public open-core GitHub release, confirm that the following surfaces are in place:

**Required docs surfaces:**
- `open-core/CHANGELOG.md` — must exist and contain the release entry being published
- `open-core/README.md` — must have a Releases section describing the public GitHub releases
- `packages/create-threadlens-app/README.md` — monorepo authoring workspace check for the npm package README
- `packages/create-threadlens-app/CHANGELOG.md` — monorepo authoring workspace check for the npm package changelog

**Artifact expectations:**
- The GitHub release on `threadlenshq/threadlens` must be tagged at the correct split SHA
- A source archive (`.tar.gz`) must be attached to the release as a downloadable artifact
- Release notes must include the changelog entry for the version being released

**Verification commands:**

```bash
gh release list --repo threadlenshq/threadlens
gh release view threadlens-vx.y.z --repo threadlenshq/threadlens
```

See the private monorepo release runbook for the full release process.

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
