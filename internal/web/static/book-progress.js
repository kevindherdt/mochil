export function bookReadingState(book) {
  const progress = book?.reading_progress
  if (!progress) return { label: '', percentage: 0, finished: false }
  if (progress.status) return { label: progress.status === 'on_hold' ? 'On hold' : 'Dropped', percentage: 0, finished: false }
  const percentage = progress.finished
    ? 100
    : Math.max(0, Math.min(100, Math.round(Number(progress.fraction || 0) * 100)))
  return {
    label: progress.finished ? 'Read' : percentage ? `${percentage}% read` : 'Started',
    percentage,
    finished: Boolean(progress.finished),
  }
}
