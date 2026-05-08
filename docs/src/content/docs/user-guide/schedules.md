---
title: Schedules
description: Use schedules for recurring scout runs after query quality is proven.
---


Schedules run scouts repeatedly so you can monitor markets over time.

## When to schedule

Add a schedule after a query has already produced useful findings in manual runs. Scheduling a noisy query creates recurring review work.

## What schedules record

ThreadLens records scout runs with platform, status, counts, and errors. Review recent runs when a scheduled workflow stops producing expected results.

## Cancellation

Long-running scout runs support graceful cancellation. Cancel a run when the query is clearly wrong or the source is returning unusable data.
