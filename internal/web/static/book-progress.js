export function bookReadingState(book) {
  const progress = book?.reading_progress
  if (!progress) return { label: '', percentage: 0, finished: false }
  // A status is a label the reader chose, and every status has its own word: an unread book used to be
  // labelled "Dropped" because only the two parked states were mapped.
  if (progress.status) {
    const labels = { on_hold: 'On hold', dropped: 'Dropped', unread: 'Unread' }
    return { label: labels[progress.status] || '', percentage: 0, finished: false }
  }
  const percentage = progress.finished
    ? 100
    : Math.max(0, Math.min(100, Math.round(Number(progress.fraction || 0) * 100)))
  return {
    label: progress.finished ? 'Read' : percentage ? `${percentage}% read` : 'Started',
    percentage,
    finished: Boolean(progress.finished),
  }
}
