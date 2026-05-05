# @scout/shared

Intentionally tiny shared package for the Scout monorepo.

## Purpose

This package holds **pure, side-effect-free** code that is clearly useful to both `apps/api` and `apps/web`. It is **not** a general-purpose utilities dumping ground.

## Rules for adding code here

Before adding anything, ask:

1. **Is it pure?** No I/O, no database, no network, no framework imports.
2. **Do both apps actually need it?** Not "might need" — actually need.
3. **Is the boundary obvious?** If you have to argue for it, it probably belongs in the app that owns it.

## Current exports

| Export | Description |
|---|---|
| `POST_STATUSES` | Tuple of all valid `posts.status` values |

## Usage

```js
import { POST_STATUSES } from '@scout/shared';
```
