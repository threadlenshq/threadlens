---
title: Ports and Local URLs
description: Reference local ThreadLens ports and URLs for Docker and development workflows.
---


| Context | URL | Notes |
| --- | --- | --- |
| Docker development web app | `http://localhost:4748` | Vite dev server. |
| Docker development Go API | `http://localhost:4749` | API container. |
| Docker production self-host app | `http://localhost:4749` | Go API serves the built web app. |
| Docs development server | `http://localhost:4750` | Astro dev server from `open-core/docs`. |
| Docs preview server | `http://localhost:4751` | Astro preview server after a docs build. |
| Public docs site | `https://docs.threadlens.dev` | Cloudflare Pages production domain. |
| Product landing page | `https://threadlens.dev` | Marketing and waitlist domain. |

Use these ports as documentation defaults unless the runtime configuration changes.
