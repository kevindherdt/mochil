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

function bookMarkup(book, layout = 'covers', showListAuthor = true) {
  if (layout === 'list') return `
    <article class="book book-list-item" data-bucket="${escapeHTML(book.bucket || '#')}" data-library="${escapeHTML(book.library_id)}" data-book="${escapeHTML(book.id)}">
      <span class="book-check" aria-hidden="true">${selectIcon}</span>
      <a class="book-open" href="${book.read_url}">
        <span class="book-list-cover"><img class="cover" decoding="async" fetchpriority="low" data-src="${escapeHTML(book.cover_url)}" data-title="${escapeHTML(book.title)}" alt=""></span>
        <span class="book-list-copy${showListAuthor ? '' : ' book-list-copy--title-only'}"><strong>${escapeHTML(book.title)}</strong>${showListAuthor ? `<small>${escapeHTML(author(book))}</small>` : ''}${readAlongMarkup(book, 'label')}${progressMarkup(book, 'label')}</span>
        ${listColumns(book, !showListAuthor)}
      </a>
      ${bookMenu(book)}
    </article>`
  return `
    <article class="book" data-bucket="${escapeHTML(book.bucket || '#')}" data-library="${escapeHTML(book.library_id)}" data-book="${escapeHTML(book.id)}">
      <span class="book-check" aria-hidden="true">${selectIcon}</span>
      ${bookMenu(book, true)}
      <a class="book-open" href="${book.read_url}" aria-label="Read ${escapeHTML(book.title)}">
        <span class="cover-frame"><img class="cover" decoding="async" fetchpriority="low" data-src="${escapeHTML(book.cover_url)}" data-title="${escapeHTML(book.title)}" alt="Cover of ${escapeHTML(book.title)}">${book.format ? `<span class="cover-tag">${escapeHTML(book.format.toUpperCase())}</span>` : ''}${readAlongMarkup(book, 'cover')}${progressMarkup(book, 'cover')}</span>
        <strong>${escapeHTML(book.title)}</strong>
      </a>
      <div class="book-meta"><small>${escapeHTML(author(book))}</small>${progressMarkup(book, 'label')}</div>
    </article>`
}


return { author, bookmarkIcon, bookMarkup, installCoverFallbacks, releaseCovers }
}
