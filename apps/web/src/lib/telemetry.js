/**
 * Browser-side telemetry client. Sends events to the Cloudflare Worker
 * via navigator.sendBeacon. Consent-gated: does nothing unless both
 * env_opt_in and ui_consent are "granted".
 */

let cachedStatus = null;
let changeListeners = new Set();

/**
 * Initializes the browser telemetry client with the status from the API.
 * Must be called once at app boot after fetching GET /api/telemetry/status.
 * @param {object} status - The response from GET /api/telemetry/status.
 */
export function initTelemetry(status) {
  cachedStatus = status || null;
}

/**
 * Updates the cached consent state and notifies all subscribers.
 * Call this after the user changes their consent choice in the UI.
 * @param {object} partial - Partial status update (e.g. { ui_consent: 'granted' }).
 */
export function updateTelemetryStatus(partial) {
  if (cachedStatus) {
    cachedStatus = { ...cachedStatus, ...partial };
  }
  changeListeners.forEach(fn => {
    try { fn(cachedStatus); } catch {}
  });
}

/**
 * Subscribes to telemetry status changes. Returns an unsubscribe function.
 * @param {function} fn - Called with the full status object on each change.
 * @returns {function} Unsubscribe function.
 */
export function onTelemetryChange(fn) {
  changeListeners.add(fn);
  return () => changeListeners.delete(fn);
}

/**
 * Returns the cached telemetry status, or null if not yet initialized.
 */
export function getTelemetryStatus() {
  return cachedStatus;
}

/**
 * Returns true if telemetry is currently allowed (both gates open).
 */
export function isTelemetryEnabled() {
  if (!cachedStatus) return false;
  const mode = cachedStatus.env_opt_in;
  if (mode === 'disabled' || !cachedStatus.instance_id) return false;
  if (mode === 'enabled') return true;
  return cachedStatus.ui_consent === 'granted';
}

/**
 * Records a single telemetry event. No-op when telemetry is disabled.
 * Uses navigator.sendBeacon for reliable delivery during page unload.
 * @param {string} eventName - One of the allow-listed event names.
 */
export function recordEvent(eventName) {
  if (!isTelemetryEnabled()) return;
  if (typeof navigator === 'undefined' || !navigator.sendBeacon) return;

  const event = {
    event_name: eventName,
    event_time_unix_ms: Date.now(),
    scout_version: cachedStatus.scout_version || '',
    deployment_type: cachedStatus.deployment_type || '',
    os_platform: cachedStatus.os_platform || '',
    source: 'client',
  };

  const batch = {
    instance_id: cachedStatus.instance_id,
    events: [event],
  };

  const blob = new Blob([JSON.stringify(batch)], { type: 'application/json' });
  navigator.sendBeacon('https://telemetry.threadlens.dev/v1/events', blob);
}

/**
 * Returns the cached instance_id, or empty string if not available.
 */
export function getInstanceId() {
  return cachedStatus?.instance_id || '';
}
