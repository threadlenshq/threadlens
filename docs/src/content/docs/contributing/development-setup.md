---
title: Development Setup
description: Set up the ThreadLens open-core workspace for local contribution.
---


Run these commands from `open-core/` unless a command explicitly says otherwise.

## Install dependencies

```bash
pnpm install
```

## Start Docker development services

```bash
pnpm run docker:dev
```

## Run the Go API without Docker

From the `open-core/` directory, start the Go API directly:

```bash
cd apps/api && go run ./cmd/scout-api
```

The API listens on `4749` unless `PORT` is set.
