export function hasCapability(snapshot, capability) {
  return Boolean(snapshot?.capabilities?.[capability]);
}

export function scoutCapabilityForPlatform(platform) {
  return `core.scout.run.${platform}`;
}

export function runtimeLabel(snapshot) {
  if (!snapshot) return 'Loading runtime';
  if (snapshot.runtimeMode === 'hosted') return 'Hosted ThreadLens';
  return 'Self-hosted ThreadLens';
}

export function isGoogleScoutAvailable(snapshot) {
  if (!snapshot) return true;
  return Boolean(snapshot?.capabilities?.['core.scout.run.google']);
}

export function isGoogleScoutLocked(snapshot) {
  if (!snapshot) return false;
  return snapshot?.capabilities?.['core.scout.run.google'] === false;
}
