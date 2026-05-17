---
title: ThreadLens Documentation
description: Self-host ThreadLens, configure providers, create a first project, scout conversations, and review results.
---

ThreadLens is open-core research intelligence for finding product opportunities in real conversations across Reddit, Bluesky, and Google Search. These docs help you run ThreadLens locally, configure providers safely, create your first project, understand scouting results, and contribute to the open-core project.

## Start with Docker

Run these commands from the `open-core/` directory:

```bash
pnpm install
pnpm run docker:dev
```

The development profile starts the Go API at `http://localhost:4749` and the web app at `http://localhost:4748`.

:::caution[Provider setup is required for real results]
Docker can start ThreadLens without provider keys, which is useful for smoke-testing that the app loads and the API responds. Useful scouting, AI scoring, analysis, and reports require at least one AI provider path, plus source-specific credentials for Google or Bluesky if you choose those sources.
:::

## First-run checklist

Follow this sequence for your first useful self-hosted run:

1. [Understand what ThreadLens does](start-here/overview/) and what the open-core release includes.
2. [Run ThreadLens with Docker](start-here/quick-start/) and verify the local web app and API are reachable.
3. [Configure provider keys](start-here/configuration-basics/) so AI workflows and your chosen scouting source can produce useful results.
4. [Reach first value in 15 minutes](start-here/first-value-in-15-minutes/) by completing setup, creating one research project, adding one narrow query, running one scout, and inspecting findings.
5. [Create your first project](start-here/first-project/) when you want the slower walkthrough with more context.
6. [Review scouting sources](user-guide/scouting-sources/) when you are ready to add Google or Bluesky credentials and compare source behavior.
7. [Generate reports](user-guide/reports/) after you have enough high-signal findings to summarize.

## Public documentation sections

The sidebar is organized for two audiences:

1. **Users** who want setup, configuration, scouting workflows, schedules, reports, and reference material.
2. **Contributors and public maintainers** who need architecture context, testing commands, docs rules, package boundaries, release hygiene, Docker image notes, and support triage.

`open-core/docs/` is publishable by default. Private hosted SaaS runbooks, credentials, billing provider details, private customer data, and proprietary roadmap notes do not belong in this docs tree.

## Related sites

- Product and waitlist: [threadlens.dev](https://threadlens.dev)
- Public repository: [github.com/threadlenshq/threadlens](https://github.com/threadlenshq/threadlens)
- License overview: [Licensing](reference/licensing/)
- Contributor setup: [Development Setup](contributing/development-setup/)
