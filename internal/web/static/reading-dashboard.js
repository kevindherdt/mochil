import { mountDashboard } from './dashboard-template.js?v=12'
import { readAlongMarkup } from './read-along-badge.js?v=2'
import { compactRelativeTime } from './date-format.js?v=1'

export function dashboardReadingTitle(item) {
  const title = String(item?.title || '').trim() || 'Untitled'
  const series = String(item?.series || '').trim()
  return series && series.toLowerCase() !== title.toLowerCase() && /^chapter\s+/i.test(title) ? `${series} — ${title}` : title
}

export function createReadingDashboard({ document, $, $$, escapeHTML, requestJSON, releaseCovers, installCoverFallbacks, isLibraryVisible = () => true }) {
mountDashboard(document)
let undoTimer
let readingFinished = false

function readingURL(item, action) {
  return `/api/reading/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}/${action}`
}

async function postReading(item, action) {
  const response = await fetch(readingURL(item, action), { method: 'POST' })
  if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`)
  return response
}

function offerUndo(message, action) {
  clearTimeout(undoTimer)
  const toast = $('#undo-toast')
  $('#undo-message').textContent = message
  $('#undo-action').onclick = async () => {
    $('#undo-action').disabled = true
    try { await action(); toast.hidden = true; await loadReadingRail(false) }
    finally { $('#undo-action').disabled = false }
  }
  toast.hidden = false
  undoTimer = setTimeout(() => { toast.hidden = true }, 8000)
}

async function loadReadingRail(showFinished = readingFinished) {
  try {
    const [activeResult, finishedResult] = await Promise.all([
      requestJSON('/api/reading?limit=4'), requestJSON('/api/reading?finished=true&limit=100'),
    ])
    const active = activeResult.data.filter(item => isLibraryVisible(item.library_id))
    const finished = finishedResult.data.filter(item => isLibraryVisible(item.library_id))
    readingFinished = showFinished && finished.length > 0
    if (!active.length && finished.length) readingFinished = true
    const items = (readingFinished ? finished : active).slice(0, 4)
    const rail = $('#reading-rail')
    rail.hidden = !active.length && !finished.length
    if (rail.hidden) return
    $('#reading-heading').textContent = readingFinished ? 'Finished books' : 'Continue reading'
    const toggle = $('#finished-toggle')
    toggle.hidden = !finished.length || (!active.length && readingFinished)
    toggle.textContent = readingFinished ? 'Continue reading' : `Finished · ${finished.length}`
    toggle.setAttribute('aria-pressed', readingFinished)
    releaseCovers($('#reading-track'))
    $('#reading-track').innerHTML = items.map((item, index) => {
      const displayTitle = dashboardReadingTitle(item)
      return `<article class="reading-item">
      <a href="${item.read_url}" aria-label="${readingFinished ? 'Open' : 'Continue'} ${escapeHTML(displayTitle)}"><img class="cover" decoding="async" data-src="${escapeHTML(item.cover_url)}" data-title="${escapeHTML(displayTitle)}" alt=""><span><strong>${escapeHTML(displayTitle)}</strong>${readAlongMarkup(item, 'label')}<small>${readingFinished ? 'Finished' : `${Math.round(item.fraction * 100)}% · ${compactRelativeTime(item.updated_at)}`}</small><i aria-hidden="true" style="--progress:${Math.round(item.fraction * 100)}%"></i></span></a>
      ${readingFinished ? `<button class="reading-restore" data-library="${escapeHTML(item.library_id)}" data-book="${escapeHTML(item.id)}">Restore</button>` : `<button class="reading-dismiss" data-index="${index}" aria-label="Remove ${escapeHTML(displayTitle)} from Continue reading"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17"/></svg></button>`}
    </article>`
    }).join('')
    installCoverFallbacks($('#reading-track'))
    $$('.reading-restore').forEach(button => button.onclick = async () => {
      button.disabled = true
      const response = await fetch(`/api/reading/${encodeURIComponent(button.dataset.library)}/${encodeURIComponent(button.dataset.book)}/restore`, { method: 'POST' })
      if (response.ok) loadReadingRail(false)
      else button.disabled = false
    })
    $$('.reading-dismiss').forEach(button => button.onclick = async () => {
      const item = items[Number(button.dataset.index)]
      button.disabled = true
      try {
        await postReading(item, 'dismiss')
        await loadReadingRail(false)
        offerUndo(`Removed “${dashboardReadingTitle(item)}”`, () => postReading(item, 'show'))
      } catch { button.disabled = false }
    })
  } catch {
    $('#reading-rail').hidden = true
  }
}


function toggle() { return loadReadingRail(!readingFinished) }
return { load: loadReadingRail, toggle }
}
