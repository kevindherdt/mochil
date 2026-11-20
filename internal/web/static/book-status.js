// Reading status from the book menu: reading, read, on hold, dropped or unread. The server keeps it per account;
// on hold, dropped and unread books drop out of Jump back in and Currently reading, and only unread and the two
// parked states keep the reader's place, which is what makes unread safe to press by accident.
export const STATUSES = [
  ['reading', 'Reading'],
  ['read', 'Read'],
  ['on_hold', 'On hold'],
  ['dropped', 'Dropped'],
  ['unread', 'Unread'],
]

export function currentStatus(book) {
  const progress = book?.reading_progress
  if (!progress) return 'unread'
  if (progress.status) return progress.status
  if (progress.finished) return 'read'
  return progress.fraction > 0 ? 'reading' : 'unread'
}

export function statusMenuHTML(book, escapeHTML) {
  const now = currentStatus(book)
  const items = STATUSES.map(([id, label]) => `<button type="button" role="menuitemradio" aria-checked="${id === now}" data-reading-status="${id}">${label}</button>`).join('')
  return `<details class="book-status-menu"><summary>Reading status</summary><div role="menu" aria-label="Reading status of ${escapeHTML(book.title)}">${items}</div></details>`
}

export function createBookStatus({ document, fetchImpl = globalThis.fetch?.bind(globalThis), reload = () => {} }) {
  document.addEventListener('click', async event => {
    const button = event.target.closest?.('[data-reading-status]')
    const article = button?.closest('.book[data-book]')
    if (!article) return
    button.disabled = true
    const response = await fetchImpl(`/api/reading/${encodeURIComponent(article.dataset.library)}/${encodeURIComponent(article.dataset.book)}/status`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: button.dataset.readingStatus }),
    }).catch(() => null)
    button.disabled = false
    button.closest('.book-more')?.removeAttribute('open')
    if (response?.ok) reload()
  })
}
