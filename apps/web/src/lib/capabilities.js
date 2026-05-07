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
