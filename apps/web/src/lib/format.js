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

// Parse a timestamp from the API. SQLite returns "YYYY-MM-DD HH:MM:SS" in UTC
// with no timezone suffix; force ISO 8601 so Date parsing is consistent across
// browsers (Safari rejects the space-separated form, and appending "Z" naively
// to an already-ISO string produces "...ZZ" → NaN). Returns NaN on bad input.
export function parseApiTimestamp(value) {
  if (!value) return NaN;
  let iso = String(value);
  if (!iso.includes('T')) iso = iso.replace(' ', 'T');
  if (!/[zZ]|[+-]\d{2}:?\d{2}$/.test(iso)) iso += 'Z';
  return new Date(iso).getTime();
}

