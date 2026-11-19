export function relativePublishedDate(value, now = Date.now()) {
  if (!value) return "";
  const date = new Date(value);
  const timestamp = date.getTime();
  if (
    Number.isNaN(timestamp) ||
    date.getUTCFullYear() < 1900 ||
    timestamp > now + 366 * 86400000
  )
    return "";
  const days = Math.round((timestamp - now) / 86400000);
  if (Math.abs(days) < 1) return "today";
  const relative = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  if (Math.abs(days) < 30) return relative.format(days, "day");
  const months = Math.round(days / 30.4375);
  if (Math.abs(months) < 12) return relative.format(months, "month");
  return relative.format(Math.round(days / 365.25), "year");
}

// A short "3h ago" style used where space is tight (dashboard cards). Unlike
// relativePublishedDate above, this only ever looks backwards in time and has
// no day/month/year buckets beyond days — the two intentionally read
// differently, but both now live here so there is one place that owns
// relative-time formatting instead of two independently-maintained copies.
export function compactRelativeTime(timestamp, now = Date.now()) {
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (seconds < 60) return "Just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}
