/**
 * @scout/shared
 *
 * Intentionally tiny. Only add code that is:
 *   1. Pure (no I/O, no DB, no network)
 *   2. Clearly useful to BOTH apps/api and apps/web
 *   3. Unlikely to create circular dependencies
 *
 * When in doubt, keep it in the app that owns it.
 */

/** All valid values for the posts.status column. */
export const POST_STATUSES = /** @type {const} */ ([
  'new',
  'drafted',
  'commented',
  'skipped',
  'reviewed',
  'starred',
  'excluded',
]);
