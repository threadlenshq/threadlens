---
title: Testing
description: Run the main ThreadLens validation commands.
---


Use focused checks before broader checks.

| Command | Run from | Purpose |
| --- | --- | --- |
| `pnpm run test:go` | repository root | Run Go API tests. |
| `pnpm run build:go` | repository root | Compile Go API packages. |
| `pnpm run docs:check` | repository root | Type-check and validate Astro docs content. |
| `pnpm run docs:build` | repository root | Build the static docs site. |

When changing docs content, run both `docs:check` and `docs:build` before requesting review.
