import { createBoundedImageLoader } from './bounded-image-loader.js?v=3'
import { bookReadingState } from './book-progress.js?v=2'
import { statusMenuHTML } from './book-status.js?v=2'
import { readAlongMarkup } from './read-along-badge.js?v=2'

export function createBookCards({ escapeHTML }) {
const bookmarkIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 4.5A1.5 1.5 0 0 1 7.5 3h9A1.5 1.5 0 0 1 18 4.5V21l-6-3.8L6 21V4.5Z"/></svg>'
const downloadIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12m-4-4 4 4 4-4M5 20h14"/></svg>'
const moreIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="5" r="1.4"/><circle cx="12" cy="12" r="1.4"/><circle cx="12" cy="19" r="1.4"/></svg>'
const selectIcon = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5"/></svg>'
const imageLoader = createBoundedImageLoader()

function bookMenu(book, overlay = false) {
  const title = escapeHTML(book.title)
  return `<details class="book-more${overlay ? ' book-more--overlay' : ''}"><summary aria-label="More options for ${title}">${moreIcon}</summary><div><button type="button" class="book-select">${selectIcon}<span>Select</span></button><button class="collection-save" data-type="book" data-library="${escapeHTML(book.library_id)}" data-book="${escapeHTML(book.id)}">${bookmarkIcon}<span>Save to collection</span></button>${statusMenuHTML(book, escapeHTML)}<a href="${book.download_url}" download>${downloadIcon}<span>Download</span></a></div></details>`
}

function author(book) {
  return (book.authors || []).join(', ') || 'Unknown author'
}

function progressMarkup(book, kind) {
  const state = bookReadingState(book)
  if (!state.label) return ''
  if (kind === 'cover') return `<span class="book-reading-bar${state.finished ? ' is-finished' : ''}" style="--progress:${state.percentage}%" aria-label="${escapeHTML(state.label)}"></span>`
  return `<small class="book-reading-label${state.finished ? ' is-finished' : ''}">${escapeHTML(state.label)}</small>`
}

// A typographic cover for books whose artwork is missing, broken, or tiny:
// the title wrapped and centred on a deterministic warm colour.
function wrapTitle(title, width = 13, maxLines = 6) {
  const lines = []
  let line = ''
  for (const word of String(title).split(/\s+/).filter(Boolean)) {
    if (line && (line + ' ' + word).length > width) { lines.push(line); line = word }
    else line = line ? line + ' ' + word : word
  }
  if (line) lines.push(line)
  if (lines.length > maxLines) { lines.length = maxLines; lines[maxLines - 1] = lines[maxLines - 1].replace(/.{0,2}$/, '…') }
  return lines.map(text => (text.length > width + 4 ? text.slice(0, width + 3) + '…' : text))
}

function fallbackCover(book) {
  const hue = [...book.title].reduce((sum, character) => sum + character.charCodeAt(0), 0) % 45 + 18
  const lines = wrapTitle(book.title)
  const top = 300 - (lines.length - 1) * 30
  const text = lines.map((line, index) => `<tspan x="200" y="${top + index * 60}">${escapeHTML(line)}</tspan>`).join('')
  return `data:image/svg+xml,${encodeURIComponent(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 600"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="hsl(${hue} 26% 42%)"/><stop offset="1" stop-color="hsl(${hue} 28% 28%)"/></linearGradient></defs><rect width="400" height="600" fill="url(#g)"/><rect x="28" y="28" width="344" height="544" rx="4" fill="none" stroke="#fff" stroke-opacity=".3"/><text text-anchor="middle" dominant-baseline="middle" fill="#fff" fill-opacity=".94" font-family="Georgia,serif" font-size="46" font-weight="600">${text}</text></svg>`)}`
}

const coverObserver = 'IntersectionObserver' in window ? new IntersectionObserver(entries => {
  entries.forEach(entry => {
    if (!entry.isIntersecting) return
    const image = entry.target
    coverObserver.unobserve(image)
    const source = image.dataset.src
    delete image.dataset.src
    if (source) { image.dataset.src = source; imageLoader.enqueue(image) }
    else image.src = fallbackCover({ title: image.dataset.title || 'Untitled' })
  })
}, { rootMargin: '500px' }) : null

function releaseCovers(root) {
  if (coverObserver) root.querySelectorAll('img[data-src]').forEach(image => coverObserver.unobserve(image))
  imageLoader.release(root)
}

function installCoverFallbacks(root = document) {
  root.querySelectorAll('.cover:not([data-fallback-ready])').forEach(image => {
    image.dataset.fallbackReady = 'true'
    image.addEventListener('error', () => { image.src = fallbackCover({ title: image.dataset.title || 'Untitled' }) }, { once: true })
    // Some books embed a thumbnail of a few pixels; it only blurs when scaled up.
    image.addEventListener('load', () => {
      if (image.naturalWidth && image.naturalWidth < 64 && !image.dataset.tiny) {
        image.dataset.tiny = 'true'
        image.src = fallbackCover({ title: image.dataset.title || 'Untitled' })
      }
    })
    if (image.dataset.src !== undefined) {
      if (coverObserver) coverObserver.observe(image)
      else image.src = image.dataset.src
    }
  })
}

// The extra columns of the list rows on wide screens: creator, series (or format) and a progress line.
// The header naming these columns is aria-hidden (it is a second, visual-only cue for sighted
// users), so each cell carries its own aria-label to keep the series and progress info available
// to screen readers.
function listColumns(book, inSeries) {
  const state = bookReadingState(book)
  const index = book.series_index != null ? ` #${Number(book.series_index)}` : ''
  const where = inSeries ? (book.series_index != null ? `Vol. ${Number(book.series_index)}` : '') : book.series ? `${book.series}${index}` : ''
  const meter = state.finished ? '<b>Read</b>' : state.percentage ? `<span class="book-list-meter-bar" style="--progress:${state.percentage}%"></span><b>${state.percentage}%</b>` : ''
  const seriesLabel = where ? `Series: ${where}` : 'No series'
  const progressLabel = state.label || 'Not started'
  return `<span class="book-list-col book-list-creator">${escapeHTML(author(book))}</span><span class="book-list-col book-list-series" aria-label="${escapeHTML(seriesLabel)}">${where ? escapeHTML(where) : '<em aria-hidden="true">—</em>'}</span><span class="book-list-col book-list-meter" aria-label="${escapeHTML(progressLabel)}">${meter}</span>`
}

// Chapter-based kinds show the chapter as the title ("Chapter 122"), so on a grid mixing
// several series (Discover, All) there was no way to tell which series a card belonged to
// without opening it. Series libraries where the title already names the work (regular books)
// don't need this: it would just repeat what "Series" already groups by.
const SERIES_NAME_KINDS = new Set(['manga', 'comics', 'lightnovels', 'light-novels'])

function seriesLine(book) {
  // The card shows where the book sits in its series, not just which series it is: "The Change · Book 1".
  // An index on its own is still worth a line, because that is the position a reader is looking for.
  // Older rows carry a series named after the book itself (a per-book folder read as a series); that is not a
  // series, so it prints nothing even before the catalogue is rescanned.
  const sameAsTitle = String(book.series || '').trim().toLowerCase() === String(book.title || '').trim().toLowerCase()
  const name = book.series && !sameAsTitle && SERIES_NAME_KINDS.has(book.library_kind) ? book.series : ''
  const index = seriesIndexLabel(book)
  // On the books grid the series *name* is not repeated under the title (the shelf already groups by series);
  // only the position is new information there. Manga and the like keep the name, because a chapter called
  // "Chapter 122" is unreadable without it.
  const nameIsNews = SERIES_NAME_KINDS.has(book.library_kind)
  const parts = [nameIsNews ? name : '', index].filter(Boolean)
  if (!parts.length) return ''
  return `<small class="book-series">${escapeHTML(parts.join(' · '))}</small>`
}

// What the card says under its cover in one word: the format or a state the cover cannot show. The cover already
// prints the reading state and the bar, so repeating it here is how a card ended up saying "Started" above
// "In progress" -- one fact, two lines, both competing for the same glance.
function bookCardTag(book) {
  if (book?.read_along) return 'Narrated'
  // A pill is only worth its line when it says something the card cannot show by itself. Progress is already the
  // strip on the cover, a finished book is the green tick, and "nobody has opened this" is the default state of a
  // shelf -- printing it on every unread card is noise, and the shelf reads cleaner with no pill at all. Only a
  // book the reader deliberately parked or dropped earns a word here.
  const status = book?.reading_progress?.status
  if (status === 'on_hold') return 'On hold'
  if (status === 'dropped') return 'Dropped'
  return ''
}

// A small green tick on the corner of a finished book's cover. 18px and no text, because "finished" is the one
// state that is binary and needs no explanation; anything louder competes with the artwork it is drawn over.
function finishedMark(book) {
  if (!bookReadingState(book).finished) return ''
  return '<span class="cover-done" role="img" aria-label="Finished"><svg viewBox="0 0 12 12" aria-hidden="true"><path d="M2.6 6.2 5 8.5 9.4 3.8" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/></svg></span>'
}

// The foot of a cover: the reading state in a word and a bar, printed over the artwork so the grid stays a
// wall of covers. A book with no progress gets no foot at all.
function coverFoot(book) {
  const state = bookReadingState(book)
  // A finished book gets the small tick instead of a full-width "Read" strip: a 100% bar on every read book turns
  // the shelf into a wall of green lines, and the tick says the same thing in 18px.
  if (state.finished) return ''
  if (!state.label) return ''
  // A book opened once sits at a fraction of a percent; a bar drawn at that width is a 1px sliver that reads as
  // a rendering fault. Under one percent the foot says "Started" and shows no bar, which is the truth.
  const bar = state.percentage >= 1
    ? `<i class="cover-foot-bar${state.finished ? ' is-finished' : ''}" style="--progress:${state.percentage}%"></i>`
    : ''
  return `<span class="cover-foot"><span class="cover-foot-label">${escapeHTML(state.label)}</span>${bar}</span>`
}

// A series index is sometimes baked into the title as a zero-padded prefix ("01 Dies the Fire"), which then
// reads as part of the name and sorts the shelf by a number. Only a zero-padded integer is treated as an index:
// "1984" and "127 Hours" are titles, "01 Dies the Fire" is not.
function displayTitle(book) {
  const raw = String(book?.title || '').trim() || 'Untitled'
  const prefixed = raw.match(/^0\d{1,2}\s*[-–—:.]?\s+(?=\S)/)
  if (!prefixed) return raw
  const rest = raw.slice(prefixed[0].length).trim()
  return rest.length >= 2 ? rest : raw
}

// The index that a title may carry as a zero-padded prefix ("01 Dies the Fire") is kept, not thrown away: it
// moves to its own label on the card, so the series position is still readable but no longer part of the name.
function seriesIndex(book) {
  // Every book row carries a series_index of 1 by default, so the stored number only means something when the
  // book actually belongs to a series. Without a series name it is noise, and 1984 is not "Book 1".
  const stored = Number(book?.series_index ?? book?.seriesIndex ?? 0)
  const seriesName = String(book?.series || '').trim()
  const sameAsTitle = seriesName.toLowerCase() === String(book?.title || '').trim().toLowerCase()
  const inSeries = seriesName !== '' && !sameAsTitle
  if (inSeries && Number.isFinite(stored) && stored > 0) return stored
  const prefixed = String(book?.title || '').match(/^0(\d{1,2})\s*[-–—:.]?\s+(?=\S)/)
  return prefixed ? Number(prefixed[1]) : 0
}

function seriesIndexLabel(book) {
  const index = seriesIndex(book)
  return index > 0 ? `Book ${index}` : ''
}

function bookMarkup(book, layout = 'covers', showListAuthor = true) {
  if (layout === 'list') return `
    <article class="book book-list-item" data-bucket="${escapeHTML(book.bucket || '#')}" data-library="${escapeHTML(book.library_id)}" data-book="${escapeHTML(book.id)}">
      <span class="book-check" aria-hidden="true">${selectIcon}</span>
      <a class="book-open" href="${book.read_url}">
        <span class="book-list-cover"><img class="cover" decoding="async" fetchpriority="low" data-src="${escapeHTML(book.cover_url)}" data-title="${escapeHTML(book.title)}" alt=""></span>
        <span class="book-list-copy${showListAuthor ? '' : ' book-list-copy--title-only'}"><strong>${escapeHTML(displayTitle(book))}</strong>${showListAuthor ? `<small>${escapeHTML(author(book))}</small>` : ''}${readAlongMarkup(book, 'label')}${progressMarkup(book, 'label')}</span>
        ${listColumns(book, !showListAuthor)}
      </a>
      ${bookMenu(book)}
    </article>`
  return `
    <article class="book" data-bucket="${escapeHTML(book.bucket || '#')}" data-library="${escapeHTML(book.library_id)}" data-book="${escapeHTML(book.id)}">
      <span class="book-check" aria-hidden="true">${selectIcon}</span>
      ${bookMenu(book, true)}
      <a class="book-open" href="${book.read_url}" aria-label="Read ${escapeHTML(displayTitle(book))}">
        <span class="cover-frame"><img class="cover" decoding="async" fetchpriority="low" data-src="${escapeHTML(book.cover_url)}" data-title="${escapeHTML(book.title)}" alt="Cover of ${escapeHTML(book.title)}">${seriesIndex(book) ? `<span class="cover-vol">Vol. ${escapeHTML(String(seriesIndex(book)))}</span>` : ''}${readAlongMarkup(book, 'cover')}${coverFoot(book)}${finishedMark(book)}</span>
        <strong>${escapeHTML(displayTitle(book))}</strong>
      </a>
      ${seriesLine(book)}
      <div class="book-meta"><small>${escapeHTML(author(book))}</small>${bookCardTag(book) ? `<span class="book-tag">${escapeHTML(bookCardTag(book))}</span>` : ''}</div>
      <div class="book-foot">${coverFoot(book)}</div>
    </article>`
}


return { author, bookmarkIcon, bookCardTag, bookMarkup, displayTitle, finishedMark, seriesIndex, seriesIndexLabel, installCoverFallbacks, releaseCovers }
}
