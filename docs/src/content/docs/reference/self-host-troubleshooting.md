---
title: Self-Host Troubleshooting
description: Fix common first-run Docker, env file, provider, and first scout issues.
---

Use this page when ThreadLens starts but the first-run path does not reach useful findings.

## Docker starts but the web app is unreachable

1. Check that all containers are running: `docker compose ps`
2. Confirm port 4748 is not already in use on your host: `lsof -i :4748`
3. Run the smoke test from `open-core/`: `pnpm run self-host:smoke`
4. If the smoke test fails, check container logs: `docker compose logs --tail=50 api web`
5. Restart the stack: `docker compose down && docker compose up -d`

## The setup wizard cannot save provider settings

1. Confirm `open-core/.env` exists and is mounted — the wizard writes values there.
2. Check that the file is writable by the Docker user: `ls -la open-core/.env`
3. If the file is missing, copy the example: `cp open-core/.env.example open-core/.env`
4. After saving env changes, restart containers so they pick up the new values: `docker compose restart`

## AI scoring or reports fail

1. Open `open-core/.env` and verify at least one AI key is present:
   - `ANTHROPIC_API_KEY` — primary provider (Claude)
   - `GEMINI_API_KEY` — fallback provider (Gemini)
2. Confirm the key is valid by testing it directly with the provider's API playground.
3. Check container logs for `401` or `invalid_api_key` errors: `docker compose logs api | grep -i "api_key\|401\|error"`
4. If using Anthropic, ensure your account has credits and the key has not been revoked.
5. Restart after correcting keys: `docker compose restart api`

## Google scouting fails

1. Confirm `PARALLEL_API_KEY` is set in `open-core/.env`.
2. Verify the key is active in your Parallel.ai dashboard.
3. Check for quota errors in the API logs: `docker compose logs api | grep -i "parallel\|quota\|429"`
4. Google scouting requires a project query with platform set to `google` — confirm at least one is enabled in the project settings.

## Bluesky scouting fails

1. Confirm both `BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` are set in `open-core/.env`.
2. `BLUESKY_HANDLE` must be your full handle (e.g. `yourname.bsky.social`).
3. `BLUESKY_APP_PASSWORD` must be an App Password created in Bluesky settings, not your login password.
4. Test authentication by checking logs after a failed scout: `docker compose logs api | grep -i "bluesky\|auth"`

## First query returns noisy results

Overly broad queries (e.g. `AI tools`) return high-volume, low-signal posts. Use a specific pain-point phrase instead.

- ❌ Too broad: `AI tools`
- ✅ More specific: `meeting notes too time consuming`

Tips for better queries:
- Phrase the query as a complaint or frustration, not a product category.
- Add a qualifier that signals intent (e.g. "too slow", "keeps breaking", "wish there was").
- Start with one focused query per platform and expand only after reviewing initial results.

## Reset first-run onboarding

If you need to restart the onboarding flow from scratch:

1. Stop the stack: `docker compose down`
2. Remove the SQLite data volume (this deletes all projects and posts): `docker volume rm threadlens_data` (replace `threadlens_data` with the actual volume name shown in `docker volume ls`)
3. Remove `open-core/.env` or clear provider keys from it.
4. Start the stack again: `docker compose up -d`
5. The setup wizard will reappear on next visit to the web app.

> **Warning:** Removing the data volume permanently deletes your SQLite database including all projects, posts, and reports. Back up `scout.db` first if you want to preserve your data.
