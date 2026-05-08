# ThreadLens Docs

Public documentation for the ThreadLens open-core project, built with [Astro Starlight](https://starlight.astro.build) and deployed to [docs.threadlens.dev](https://docs.threadlens.dev) via Cloudflare Pages.

## What's here

Docs covering self-hosting, configuration, the user guide, architecture internals, and contributing guidelines. Content lives in `src/content/docs/` organised by section. All content is public-only; no hosted/SaaS-specific material belongs here.

## Local development

```bash
# From the repo root (pnpm workspace)
pnpm --filter @threadlens/docs dev
# or from this directory:
pnpm dev          # Astro dev server → http://localhost:4750
pnpm build        # Production build → dist/
pnpm preview      # Preview the build → http://localhost:4751
pnpm check        # TypeScript / Astro type-check
```

## Deployment

The site deploys to **Cloudflare Pages** (`threadlens-docs` project).

### CI/CD (recommended)

This repo includes a GitHub Actions workflow at `.github/workflows/docs-cloudflare-pages.yml`.

- Trigger: pushes that touch `open-core/docs/**`
- Build: `pnpm --dir open-core run docs:build`
- Deploy: `wrangler pages deploy dist --project-name threadlens-docs`

Required repository secrets:

- `CLOUDFLARE_ACCOUNT_ID`
- `CLOUDFLARE_API_TOKEN` (with Cloudflare Pages edit/deploy permissions)

### Manual deploy

```bash
pnpm deploy       # astro build + wrangler pages deploy dist
```

The `wrangler.toml` in this directory targets the `threadlens-docs` Cloudflare Pages project.

## Adding docs

1. Drop a `.md` or `.mdx` file in the relevant `src/content/docs/<section>/` directory.
2. Add a `slug` entry to the sidebar in `astro.config.mjs` if it's a new page.
3. Run `pnpm dev` to verify locally before opening a PR.

See `src/content/docs/contributing/docs-contributions.md` for full guidelines.
