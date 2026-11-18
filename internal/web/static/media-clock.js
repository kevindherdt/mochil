/** Shared catalogue/player clock with compact hour-aware output. */
export function mediaClock(seconds, empty = '0:00') {
  const total = Math.floor(Number(seconds))
  if (!Number.isFinite(total) || total <= 0) return empty
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor(total % 3600 / 60)
  const remainder = total % 60
  return hours
    ? `${hours}:${String(minutes).padStart(2, '0')}:${String(remainder).padStart(2, '0')}`
    : `${minutes}:${String(remainder).padStart(2, '0')}`
}
