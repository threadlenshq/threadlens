---
title: Testing Commands
description: Run the main ThreadLens open-core validation commands.
---

# Testing Commands

Use focused checks before broader checks.

| Command | Run from | Purpose |
| --- | --- | --- |
| `pnpm run test:go` | `open-core/` | Run Go API tests. |
| `pnpm run build:go` | `open-core/` | Compile Go API packages. |
| `pnpm run docs:check` | `open-core/` | Type-check and validate Astro docs content. |
| `pnpm run docs:build` | `open-core/` | Build the static docs site. |

When changing docs content, run both `docs:check` and `docs:build` before requesting review.
