import { headphones, readAlongMarkup } from './read-along-badge.js?v=2'
import { dashboardReadingTitle } from './reading-dashboard.js?v=18'

const LABELS = { books: 'Book', lightnovels: 'Light novel', 'light-novels': 'Light novel', novels: 'Light novel', manga: 'Manga', comics: 'Comic', movies: 'Movie', tv: 'TV show', video: 'Video', anime: 'Anime', music: 'Music', audiobooks: 'Audiobook', podcasts: 'Podcast' }

import { kindIconSVG } from './kind-icons.js?v=2'
import { libraryHue } from './library-identity.js?v=2'

export const kindIcon = item => kindIconSVG(item.library_kind)

export const kindLabel = item => LABELS[item.library_kind] || (item.type === 'book' ? 'Book' : 'Media')

/** The colour family of a card: books and their kin share one, and so on. Styled by data-family, so a new kind only needs a line here. */
export const kindFamily = ({ type, item }) => type === 'book' ? 'book' : type === 'video' ? 'video' : item.library_kind === 'podcasts' ? 'podcast' : 'audio'

const clock = seconds => {
  const total = Math.max(0, Math.round(seconds)), hours = Math.floor(total / 3600), minutes = Math.floor(total % 3600 / 60)
  return hours ? `${hours} h ${minutes} min` : `${Math.max(1, minutes)} min`
}

// Playback progress is stamped in seconds (video.Progress.UpdatedAt and
// audio.Progress.UpdatedAt are time.Now().Unix) while a reading position is stamped in
// milliseconds (reading.Position.UpdatedAt is time.Now().UnixMilli). Ordering the two
// raw numbers put every book ahead of an episode watched a minute ago, however stale the
// book was, so media stamps are widened before the sort.
const mediaAt = value => (value > 0 ? value * 1000 : 0)

/** One list, newest activity first, of what is in progress across books, video and audio (songs are not "continued"). */
export function mergeJumpItems({ books = [], videos = [], audios = [] }, isLibraryVisible = () => true, limit = 8, perType = 4) {
  const entries = []
  for (const item of books) {
    if (item.updated_at && !item.finished && !item.dismissed && isLibraryVisible(item.library_id)) entries.push({ type: 'book', item, at: item.updated_at })
  }
  for (const item of videos) {
    const progress = item.progress
    if (progress && !progress.finished && !progress.dismissed && isLibraryVisible(item.library_id) && (item.stream_url || item.hls_url)) entries.push({ type: 'video', item, at: mediaAt(progress.updated_at) })
  }
  for (const item of audios) {
    const progress = item.progress
    if (progress && !progress.finished && !progress.dismissed && item.library_kind !== 'music' && isLibraryVisible(item.library_id)) entries.push({ type: 'audio', item, at: mediaAt(progress.updated_at) })
  }
  // No kind may crowd out the others: keep the newest few of each, then order them together.
  const taken = { book: 0, video: 0, audio: 0 }
  return entries.sort((a, b) => b.at - a.at).filter(entry => taken[entry.type]++ < perType).slice(0, limit)
}

/** How far along, in the words that fit the kind of thing it is. */
export function jumpDetail({ type, item }) {
  if (type === 'book' && item.status) return item.status === 'on_hold' ? 'On hold' : 'Dropped'
  if (type === 'book') return item.finished ? 'Finished' : `${Math.round((item.fraction || 0) * 100)}% read`
  const { position_seconds: position = 0, duration_seconds: duration = 0 } = item.progress
  return duration > position ? `${clock(duration - position)} left` : 'In progress'
}

export function jumpCardHTML(entry, escapeHTML) {
  const { type, item } = entry
  const title = type === 'book' ? dashboardReadingTitle(item) : item.title || 'Untitled'
  const who = type === 'book' ? (item.authors || []).join(', ') : item.artist || item.series || ''
  const art = type === 'book'
    ? `<img class="cover" decoding="async" data-src="${escapeHTML(item.cover_url || '')}" data-title="${escapeHTML(title)}" alt="">`
    : item.artwork_url ? `<img class="cover" loading="lazy" decoding="async" src="${escapeHTML(item.artwork_url)}" alt="">` : ''
  // The backdrop is a blurred copy of the art. A book's art is the full-size cover the card itself
  // never loads (the grid asks for a sized one), so it cost one extra full-size fetch per card and
  // bought only blur; books keep the plain raised background. Video and podcast artwork is already
  // the URL their own image loads, so their backdrop is free.
  const backdrop = type !== 'book' && art ? ` style="--art:url('${encodeURI(item.artwork_url).replaceAll("'", '%27').replaceAll('(', '%28').replaceAll(')', '%29')}')"` : ''
  const open = type === 'book'
    ? `<a class="jump-card" href="${escapeHTML(item.read_url)}" data-family="${kindFamily(entry)}"`
    : `<button type="button" class="jump-card" data-jump-type="${type}" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}" data-family="${kindFamily(entry)}"`
  const close = type === 'book' ? 'a' : 'button'
  // The headphones icon alone is too subtle: a read-along book can otherwise sit next to its own
  // plain EPUB progress card with no visible difference, and look like an accidental duplicate.
  // A short text label next to the title makes the two kinds of card tell themselves apart.
  return `${open} data-kind="${escapeHTML(item.library_kind || type)}"><span class="jump-art"${backdrop}>${art}<span class="jump-tag" role="img" aria-label="${escapeHTML(kindLabel(item))}" title="${escapeHTML(kindLabel(item))}" style="--kind-h:${libraryHue(item.library_id)}">${kindIcon(item)}${type === 'book' && item.read_along ? `<span class="jump-tag-audio" title="Has its own recording" role="img" aria-label="Has its own recording">${headphones}</span>` : ''}</span></span><strong>${escapeHTML(title)}</strong><span class="jump-badge">${type === 'book' ? readAlongMarkup(item, 'label') : ''}</span><small>${escapeHTML([who, jumpDetail(entry)].filter(Boolean).join(' · '))}</small></${close}>`
}

/** How to hide (or bring back) one in-progress item: each kind has its own endpoint. */
export function dismissRequest(type, item, dismissed) {
  const where = `${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}`
  if (type === 'book') return { url: `/api/reading/${where}/${dismissed ? 'dismiss' : 'show'}`, options: { method: 'POST' } }
  return { url: `/api/${type === 'video' ? 'video' : 'audio'}-progress/${where}`, options: { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ dismissed }) } }
}

/** A card plus the small button that takes it off the row. */
export function jumpItemHTML(entry, escapeHTML) {
  const title = escapeHTML(entry.type === 'book' ? dashboardReadingTitle(entry.item) : entry.item.title || 'Untitled')
  return `<article class="jump-item">${jumpCardHTML(entry, escapeHTML)}<button type="button" class="jump-dismiss" data-jump-dismiss data-jump-type="${entry.type}" data-library="${escapeHTML(entry.item.library_id)}" data-item="${escapeHTML(entry.item.id)}" data-title="${title}" aria-label="Remove ${title} from Jump back in" title="Remove"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg></button></article>`
}

/** The row on Home that mixes everything in progress. Opening a video or audio card presses the matching card in the lanes below, which already know how to start it. */
export function createJumpBackIn({ document, $, escapeHTML, requestJSON, isLibraryVisible, releaseCovers, installCoverFallbacks }) {
  let request = 0, undoTimer
  let shown = new Map()
  // What the home screen says when there is nothing to jump back into. On a brand-new install the
  // honest answer is not "media will appear here" but "nothing is configured yet".
  async function emptyMessage() {
    const libraries = await requestJSON("/api/libraries").then(
      (result) => (Array.isArray(result?.data) ? result.data : []),
      () => [],
    )
    const items = libraries.reduce((total, library) => total + (Number(library.count) || 0), 0)
    return items === 0
      ? 'No books here yet. Put files in your books folder, or <a href="#/settings">add a folder in Settings</a>.'
      : null
  }

  async function load() {
    const mine = ++request
    const section = $('#jump-rail'), track = $('#jump-track')
    if (!section || !track) return
    const get = url => requestJSON(url).then(result => (Array.isArray(result?.data) ? result.data : []), () => [])
    const [books, videos, audios] = await Promise.all([
      get('/api/reading?limit=6'), get('/api/video-progress'), get('/api/audio-progress'),
    ])
    if (mine !== request) return
    const entries = mergeJumpItems({ books, videos, audios }, isLibraryVisible)
    section.hidden = entries.length < 1
    const empty = $('#dashboard-empty')
    if (empty) {
      empty.hidden = !section.hidden
      // Only when the message is actually on screen: nothing to jump back into.
      if (!empty.hidden) {
        const firstRun = await emptyMessage()
        if (firstRun) empty.innerHTML = firstRun
        else empty.textContent = 'Media you start will wait for you here.'
      }
    }
    releaseCovers(track)
    track.innerHTML = section.hidden ? '' : entries.map(entry => jumpItemHTML(entry, escapeHTML)).join('')
    installCoverFallbacks(track)
    shown = new Map(entries.map(entry => [`${entry.type}/${entry.item.library_id}/${entry.item.id}`, entry]))
  }

  async function dismiss(button) {
    const { jumpType, library, item, title } = button.dataset
    const entry = shown.get(`${jumpType}/${library}/${item}`)
    if (!entry) return
    const send = async dismissed => {
      const { url, options } = dismissRequest(jumpType, entry.item, dismissed)
      const response = await fetch(url, options)
      if (!response.ok) throw new Error(`HTTP ${response.status}`)
      await load()
    }
    button.disabled = true
    try {
      await send(true)
      const toast = $('#undo-toast')
      $('#undo-message').textContent = `Removed “${title}”`
      $('#undo-action').onclick = async () => { await send(false); toast.hidden = true }
      toast.hidden = false
      clearTimeout(undoTimer)
      undoTimer = setTimeout(() => { toast.hidden = true }, 8000)
    } catch { button.disabled = false }
  }
  document.addEventListener('click', event => {
    const remove = event.target.closest?.('[data-jump-dismiss]')
    if (remove) { void dismiss(remove); return }
    const card = event.target.closest?.('.jump-card[data-jump-type]')
    if (!card) return
    const { library, item, jumpType } = card.dataset
    const match = `[data-library="${CSS.escape(library)}"][data-item="${CSS.escape(item)}"]`
    document.querySelector(jumpType === 'video' ? `.video-reading-item${match}` : `.audio-item${match} .audio-open`)?.click()
  })
  return { load }
}
