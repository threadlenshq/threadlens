---
title: Local AI Bridge
description: Use the optional local host CLI bridge for Docker development and advanced self-hosted CLI reuse.
---

The local AI bridge is an optional helper that lets the Dockerized or self-hosted API reuse Copilot CLI and Claude CLI sessions that are already authenticated on the host machine. It is not required for normal Docker startup, and it is not part of the production baseline.

## Normal Docker behavior

For most users, `pnpm run docker:dev` is enough.

On macOS and Linux, the Docker dev bootstrap automatically:

- creates `~/.config/scout/ai-bridge.token` if needed
- writes `~/.config/scout/ai-bridge.json`
- tries to build and start `scout-ai-bridge`
- enables Docker bridge env vars only when the bridge is healthy and has at least one available runtime

If the bridge is unavailable, Docker still starts normally. ThreadLens then falls back to configured API-key providers or direct in-runtime CLI providers.

## When to use the manual helper

Use the manual helper when you want to:

- start or stop the bridge outside Docker bootstrap
- inspect bridge health directly
- pre-build the bridge before running Docker
- run the bridge for advanced self-hosted setups

Run these commands from the public repository root:

```bash
pnpm run bridge:bootstrap
pnpm run bridge:start
pnpm run bridge:status
pnpm run bridge:health
pnpm run bridge:stop
```

## What each command does

| Command | Purpose |
| --- | --- |
| `pnpm run bridge:bootstrap` | Creates the token/config files and builds the bridge binary if needed. Does not start the daemon. |
| `pnpm run bridge:start` | Bootstraps and starts the local bridge daemon in the background. |
| `pnpm run bridge:status` | Shows whether the managed bridge process is running and whether health is passing. |
| `pnpm run bridge:health` | Prints the raw bridge health JSON. |
| `pnpm run bridge:stop` | Stops the managed bridge daemon started by the helper. |

## Requirements

- macOS or Linux
- `go` installed if `bin/scout-ai-bridge` has not been built yet
- `copilot` and/or `claude` installed on the host if you want CLI-backed models through the bridge
- at least one of those CLIs authenticated on the host

For installation and authentication steps, see [Credential Setup — GitHub Copilot CLI](credential-setup/#copilot-cli) and [Credential Setup — Claude CLI](credential-setup/#claude-cli).

## Health expectations

When healthy, `pnpm run bridge:health` returns JSON like:

```json
{
  "ok": true,
  "runtimes": [
    {
      "id": "copilot",
      "available": true,
      "message": "ok"
    }
  ]
}
```

An empty runtime list means the bridge is reachable but no usable host CLI runtime is currently available.

## Safety

- The bridge binds to `127.0.0.1:4761` by default.
- Do not expose it on a public interface.
- The bridge token is local secret material; do not commit or publish it.

For the main provider flow, see [Model and Provider Configuration](../user-guide/model-provider-configuration/). For Docker startup behavior, see [Docker Commands and Profiles](./docker-commands-and-profiles/).
