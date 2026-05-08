---
title: Data Storage and Backups
description: Understand where ThreadLens stores local data and how to think about backups.
---

# Data Storage and Backups

ThreadLens stores open-core application data in SQLite.

## Docker storage

Docker profiles store SQLite data in the named volume `scout_open_core_sqlite_data`. The API container sees the database at `/data/scout.db`.

## Local API storage

When running the Go API outside Docker, `SCOUT_DB_PATH` controls the SQLite path. If `SCOUT_DB_PATH` is empty, the API uses the default path from its configuration.

## Backup notes

- Stop write-heavy local activity before copying a SQLite database file.
- Back up the Docker volume before deleting it.
- Keep provider keys in `.env`; do not put secrets into database backups shared for bug reports.
