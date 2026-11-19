import { applyCoverTint } from './cover-tint.js?v=3'
import { bookReadingState } from './book-progress.js?v=2'

const BACK = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 5-7 7 7 7"/></svg>'

const volumeLabel = book => book.series_index != null ? `Vol. ${Number(book.series_index)}` : ''

/** Which book the big button opens: what is being read right now, otherwise the first one not yet finished. */
export function seriesContinueTarget(books) {
  const states = books.map(book => ({ book, state: bookReadingState(book) }))
  const reading = states.filter(({ state }) => state.percentage > 0 && !state.finished)
    .sort((a, b) => String(b.book.reading_progress?.updated_at || '').localeCompare(String(a.book.reading_progress?.updated_at || '')))[0]
  if (reading) return { book: reading.book, mode: 'continue' }
  const next = states.find(({ state }) => !state.finished)
  if (next) return { book: next.book, mode: 'start' }
  return books.length ? { book: books[0], mode: 'again' } : null
}

export function seriesStats(books, total) {
  const read = books.filter(book => bookReadingState(book).finished).length
  const known = Math.max(total || 0, books.length)
  return { read, total: known, percent: known ? Math.round(read / known * 100) : 0 }
}

export function seriesActionLabel(target) {
  if (!target) return ''
  const vol = volumeLabel(target.book)
  if (target.mode === 'continue') return vol ? `Continue ${vol}` : 'Continue reading'
  if (target.mode === 'start') return vol ? `Start ${vol}` : 'Start reading'
  return 'Read again'
}

/** The top of one series: cover, tags, title, author, how far along you are, and the button that picks up where you left off. */
export function seriesHubHTML(item, { escapeHTML, authors, saveIcon }) {
  const art = item.cover_url ? ` style="--art:url('${encodeURI(item.cover_url).replaceAll("'", '%27').replaceAll('(', '%28').replaceAll(')', '%29')}')"` : ''
  const count = item.count.toLocaleString()
  return `<header class="series-hub"${art}>
    <button class="back-button" aria-label="Back to series">${BACK}</button>
    <div class="series-hub-body">
      <span class="series-hub-cover"><img class="cover" decoding="async" data-src="${escapeHTML(item.cover_url || '')}" data-title="${escapeHTML(item.name)}" alt="Cover of ${escapeHTML(item.name)}"><b>${count} ${item.count === 1 ? 'BOOK' : 'VOLUMES'}</b></span>
      <div class="series-hub-copy">
        <div class="series-hub-tags"><span>Series</span></div>
        <h2>${escapeHTML(item.name)}</h2>
        ${authors ? `<p class="series-hub-by">By <strong>${escapeHTML(authors)}</strong></p>` : ''}
        <p class="series-hub-progress" id="series-hub-progress"></p>
        <div class="series-hub-actions">
          <a class="series-hub-primary" id="series-hub-continue" href="#" hidden></a>
          <button class="collection-save series-save series-hub-save" data-type="series" data-library="${escapeHTML(item.library_id)}" data-series="${escapeHTML(item.name)}" aria-label="Save ${escapeHTML(item.name)} to collection">${saveIcon}<span>Save to collection</span></button>
        </div>
      </div>
    </div>
  </header>`
}

/** Keeps the hub's progress line and button in step with the books the page has loaded so far. */
export function bindSeriesHub(root, total) {
  applyCoverTint(root.querySelector('.series-hub'), root.querySelector('.series-hub-cover img'))
  const books = []
  const progress = root.querySelector('#series-hub-progress'), button = root.querySelector('#series-hub-continue')
  function update() {
    const stats = seriesStats(books, total)
    progress.textContent = stats.read ? `${stats.read} of ${stats.total} read (${stats.percent}% complete)` : ''
    const target = seriesContinueTarget(books), label = seriesActionLabel(target)
    button.hidden = !target
    if (target) { button.href = target.book.read_url; button.textContent = label }
  }
  return page => { books.push(...page); update() }
}
