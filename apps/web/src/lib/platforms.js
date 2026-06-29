export const PLATFORM_LABELS = {
  reddit: 'Reddit',
  bluesky: 'Bluesky',
  google: 'Google',
  hackernews: 'Hacker News',
};

export function platformLabel(platform) {
  return PLATFORM_LABELS[platform] ?? platform;
}

export const PLATFORM_COLORS = {
  reddit: '#FF4500',
  bluesky: '#0085FF',
  hackernews: '#FF6600',
};
