---
title: Workspace Layout
description: Understand the main directories in the ThreadLens open-core workspace.
---


The open-core workspace is rooted at `open-core/`.

```text
apps/
  web/             Svelte 5 and Vite frontend for the local app
  api/             Go backend and API server
packages/
  shared/          Shared pure code used by web and API packages
infra/
  docker/          Docker Compose and container build configuration
db/                Shared database configuration and migration support
docs/              Astro and Starlight documentation site
```

Keep source files close to the code they explain. Durable public documentation belongs in `open-core/docs/`; short entry-point notes can stay in package READMEs and link to canonical docs pages.
