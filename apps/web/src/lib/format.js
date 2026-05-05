export function scoreColor(score) {
  if (score >= 9) return '#e5c07b';
  if (score >= 7) return '#98c379';
  if (score >= 5) return '#61afef';
  if (score >= 3) return '#d19a66';
  return '#6b6b80';
}

export function formatDate(dateStr) {
  if (!dateStr) return '';
  return new Date(dateStr).toLocaleDateString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
