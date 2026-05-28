---
title: Credential Setup
description: How to obtain each credential ThreadLens uses for AI providers, search, and Bluesky scouting.
---

This page covers where to get each credential ThreadLens uses. Each section links to the official provider page and notes any prerequisites or subscription requirements.

## Anthropic API key

`ANTHROPIC_API_KEY` — enables Anthropic-backed AI scoring, analysis, clustering, and reports.

1. Sign in or create an account at [console.anthropic.com](https://console.anthropic.com).
2. Navigate to **Settings → API keys** or go directly to [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys).
3. Click **Create key**, give it a name, and copy the key value.
4. Add it to your `.env` file:

   ```dotenv
   ANTHROPIC_API_KEY=sk-ant-...
   ```

Billing is usage-based. Review the [Anthropic pricing page](https://www.anthropic.com/pricing) before enabling high-volume scouting.

## Gemini API key

`GEMINI_API_KEY` — enables the Gemini-compatible provider path for AI scoring, analysis, and reports.

1. Sign in or create a Google account, then open [Google AI Studio](https://aistudio.google.com/apikey).
2. Click **Create API key** and select or create a Google Cloud project.
3. Copy the key value.
4. Add it to your `.env` file:

   ```dotenv
   GEMINI_API_KEY=AIza...
   ```

## GitHub Copilot CLI

`copilot` provider — no environment variable required. Availability is detected at runtime by binary presence.

**Prerequisite:** A GitHub Copilot subscription (Individual, Business, or Enterprise). See [GitHub Copilot plans](https://github.com/features/copilot) for options.

1. Install the GitHub CLI (`gh`) from [cli.github.com](https://cli.github.com) or via your package manager:

   ```bash
   # macOS
   brew install gh

   # Linux (apt)
   sudo apt install gh
   ```

2. Authenticate the GitHub CLI:

   ```bash
   gh auth login
   ```

3. Install the GitHub Copilot CLI extension:

   ```bash
   gh extension install github/gh-copilot
   ```

4. Verify by running:

   ```bash
   gh copilot --version
   ```

For full setup details see the [GitHub Copilot in the CLI documentation](https://docs.github.com/en/copilot/how-tos/use-copilot-for-common-tasks/use-copilot-in-the-cli).

No `.env` change is needed. ThreadLens detects `copilot` availability through the local AI bridge if running in Docker, or directly if running outside Docker.

## Claude CLI

`claude-cli` provider — no environment variable required. Availability is detected at runtime by binary presence.

1. Download and install Claude Code from [claude.ai/download](https://claude.ai/download).
2. Follow the installation steps for your platform.
3. Authenticate by running:

   ```bash
   claude
   ```

   The CLI will prompt you to log in with your Anthropic account on first launch.

4. Verify the CLI is available:

   ```bash
   claude --version
   ```

For full setup details see the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code/getting-started).

No `.env` change is needed. ThreadLens detects `claude-cli` availability through the local AI bridge if running in Docker, or directly if running outside Docker.

## Bluesky credentials

`BLUESKY_HANDLE` and `BLUESKY_APP_PASSWORD` — both required to enable Bluesky scouting.

**`BLUESKY_HANDLE`** is your full Bluesky handle, for example `yourname.bsky.social`. It is not a secret.

**`BLUESKY_APP_PASSWORD`** is a Bluesky App Password, not your login password. App Passwords are scoped credentials that can be revoked independently.

To create an App Password:

1. Sign in to [bsky.app](https://bsky.app).
2. Go to **Settings → Privacy and Security → App Passwords** or open [bsky.app/settings/app-passwords](https://bsky.app/settings/app-passwords) directly.
3. Click **Add App Password**, give it a name such as `threadlens`, and copy the generated password.
4. Add both values to your `.env` file:

   ```dotenv
   BLUESKY_HANDLE=yourname.bsky.social
   BLUESKY_APP_PASSWORD=xxxx-xxxx-xxxx-xxxx
   ```

Do not use your account login password. Only App Passwords are supported, and they can be revoked from the same settings page without affecting your account.

## Parallel API key

`PARALLEL_API_KEY` — enables Google Search scouting through the Parallel.ai search provider.

1. Create an account at [platform.parallel.ai](https://platform.parallel.ai).
2. Navigate to **API keys** and create a new key.
3. Copy the key value.
4. Add it to your `.env` file:

   ```dotenv
   PARALLEL_API_KEY=par-...
   ```

For API reference and search usage see [docs.parallel.ai](https://docs.parallel.ai/getting-started/overview).

## Safe example values

Use obviously fake values in documentation, examples, screenshots, and bug reports:

```dotenv
ANTHROPIC_API_KEY=sk-ant-example-not-real
GEMINI_API_KEY=gemini-example-not-real
PARALLEL_API_KEY=parallel-example-not-real
BLUESKY_HANDLE=example.bsky.social
BLUESKY_APP_PASSWORD=example-app-password-not-real
```

Do not commit real keys, passwords, or tokens to version control.

## Related pages

- [Environment Variables](environment-variables/) — complete variable reference
- [Configuration Basics](../start-here/configuration-basics/) — first-run setup guide
- [Model and Provider Configuration](../user-guide/model-provider-configuration/) — provider fallback behavior
- [Local AI Bridge](local-ai-bridge/) — using CLI providers through Docker
