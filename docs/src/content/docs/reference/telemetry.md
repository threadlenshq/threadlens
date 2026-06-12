---
title: Anonymous Telemetry
description: Understand what ThreadLens telemetry collects, how to enable it, and how to opt out.
---

ThreadLens includes an optional, anonymous telemetry pipeline that helps the maintainers understand how the self-hosted product is used. Telemetry is **off by default** and requires two explicit opt-in gates before any data leaves your instance.

## Two-gate consent model

Telemetry only flows when **both** of these conditions are true:

1. **Infrastructure opt-in:** The environment variable `SCOUT_TELEMETRY_OPT_IN` is set to `1`. Without this, the telemetry client is a no-op regardless of any UI setting.
2. **UI consent:** The user has explicitly accepted telemetry in the onboarding wizard, the bottom-left consent toast, or the Settings → Privacy page.

If either gate is closed, no events are sent.

## What is collected

| Category | Event name | When it fires |
| --- | --- | --- |
| Heartbeat | `instance_started` | Once per API process start |
| Heartbeat | `instance_ping` | Once every 24 hours |
| Heartbeat | `instance_consent_changed` | Once per consent change |
| Feature usage | `feature_used:scout_run` | When a scout run is started |
| Feature usage | `feature_used:query_suggest` | When query suggestions are requested |
| Feature usage | `feature_used:report_create` | When a report is generated |
| Feature usage | `feature_used:schedule_create` | When a schedule is created |
| Feature usage | `feature_used:filter_job` | When a filter job is created |
| Errors | `error:onboarding_save` | When onboarding save returns a server error |
| Errors | `error:scout_run` | When a scout run ends in failed status |

Every event includes: `event_name`, `event_time_unix_ms`, `scout_version`, `deployment_type` (docker/local), `os_platform` (linux/darwin/windows/unknown), and `source` (server/client).

A random `instance_id` (UUID) is generated on first launch and stored in the local database. This lets the team count instances, not people.

## What is never collected

- Personal data, usernames, or email addresses
- Query text, prompt content, post content, or report content
- API keys, environment variable values, or file paths
- Hostnames, IP addresses, MAC addresses, or container IDs
- Error messages, stack traces, or HTTP request bodies
- Project IDs, project names, or record counts

## How to enable telemetry

Add the following to your `.env` file and restart Docker:

```bash
SCOUT_TELEMETRY_OPT_IN=1
```

After restarting, the onboarding wizard (for new installs) or a bottom-left toast (for existing installs) will prompt for consent.

## How to opt out

- **From the UI:** Go to Settings → Privacy and toggle the consent switch off.
- **From the environment:** Remove or unset `SCOUT_TELEMETRY_OPT_IN` from your `.env` file and restart Docker. This is the infrastructure-level kill switch and overrides any UI setting.

## Where data goes

Events are sent via HTTPS to `telemetry.threadlens.dev`, a Cloudflare Worker that writes to Cloudflare Analytics Engine. The worker validates every event against a strict schema and rejects anything outside the allow-list.

## Data retention

Cloudflare Analytics Engine retains data for 90 days by default. No telemetry data is exported to third-party analytics services.

## Source code

The telemetry pipeline is fully open-source:

- **Go API recorder:** `open-core/apps/api/internal/telemetry/`
- **Browser client:** `open-core/apps/web/src/lib/telemetry.js`
- **Worker source:** `infra/cloudflare/telemetry-worker/`
