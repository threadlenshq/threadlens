---
title: Package Boundary Expectations
description: Keep ThreadLens open-core code organized by clear package responsibilities.
---

# Package Boundary Expectations

Respect the existing open-core package boundaries.

## `apps/api`

Use the Go API for backend routes, services, repositories, pipelines, runtime configuration, and scheduling.

## `apps/web`

Use the Svelte web app for local UI, client-side state, API wrappers, and visual interaction.

## `packages/shared`

Only add pure, side-effect-free code that both `apps/api` and `apps/web` actually need.

## `docs`

Use the docs package for public documentation source, Starlight config, Cloudflare Pages config, and docs-specific styling.
