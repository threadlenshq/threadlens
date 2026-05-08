---
title: ThreadLens Documentation
description: Run, configure, understand, and contribute to the ThreadLens open-core research intelligence app.
---

ThreadLens is open-core research intelligence for finding product opportunities in real conversations across Reddit, Bluesky, and Google Search. These docs help you run ThreadLens locally, configure providers safely, understand how scouting works, and contribute to the open-core project.

## Start with Docker

Run these commands from the `open-core/` directory:

```bash
pnpm install
pnpm run docker:dev
```

The development profile starts the Go API at `http://localhost:4749` and the web app at `http://localhost:4748`.

## Choose your path

- [What ThreadLens is](start-here/overview/) explains the product, audience, and current open-core status.
- [Run ThreadLens with Docker](start-here/quick-start/) gives the fastest local setup path.
- [Configure provider keys](start-here/configuration-basics/) explains the `.env` file and safe sample values.
- [Create your first project](start-here/first-project/) walks through queries, scouting, findings, and reports.
- [Contribute to ThreadLens](contributing/development-setup/) explains local development and quality checks.

## Public documentation sections

The sidebar is organized for two audiences:

1. **Users** who want setup, configuration, scouting workflows, schedules, reports, and reference material.
2. **Contributors and public maintainers** who need architecture context, testing commands, docs rules, package boundaries, release hygiene, Docker image notes, and support triage.

`open-core/docs/` is publishable by default. Private hosted SaaS runbooks, credentials, billing provider details, private customer data, and proprietary roadmap notes do not belong in this docs tree.

## Related sites

- Product and waitlist: [threadlens.dev](https://threadlens.dev)
- Public repository: [github.com/threadlenshq/threadlens](https://github.com/threadlenshq/threadlens)
