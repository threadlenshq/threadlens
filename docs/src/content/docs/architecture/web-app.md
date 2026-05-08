---
title: Web App Architecture
description: Understand the open-core Svelte and Vite frontend role.
---

# Web App Architecture

The open-core frontend lives in `open-core/apps/web` and uses Svelte 5 with Vite.

## Responsibilities

- Render project selection, query editing, posts, reports, schedules, and runtime banners.
- Call the Go API through fetch wrappers in `src/lib/api.js`.
- Preserve URL state for selected project, filters, and view state.
- Use Vite development proxying in Docker development so browser `/api` requests reach the Go API.

## Build relationship

The production Docker profile builds the frontend and serves the static output through the Go API image.
