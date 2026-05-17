---
title: Issue Reproductions
description: Report useful ThreadLens open-core issues without sharing secrets.
---


Good bug reports make open-core maintenance faster and safer.

## Include

- The command you ran.
- The directory you ran it from.
- Docker profile if Docker was involved.
- Whether `pnpm run self-host:smoke` passed or the first line it reported as unreachable.
- Whether the setup wizard completed.
- Whether a project and first query were created.
- Source being scouted, such as Reddit, Bluesky, or Google.
- The first query text if it is not private.
- Redacted error output.
- Whether the issue reproduces with fake or empty provider keys.

## Exclude

- Real `.env` files.
- Provider keys.
- Private URLs.
- Customer or account data.
- Hosted SaaS credentials.
