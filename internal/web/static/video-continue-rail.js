
/** How far into a film or episode: a percentage reads true even in the first minute, where "0 min watched" read
 *  as a mistake on a rail that exists to resume things. */
export function watchedLabel(progress) {
  const { position_seconds: position = 0, duration_seconds: duration = 0 } = progress
  if (duration > 0) {
    const percent = Math.min(99, Math.max(1, Math.round(position / duration * 100)))
    return `${percent}% watched`
  }
  return 'In progress'
}

import { cleanVideoTitle } from './video-title.js?v=1'
import { setRailCount } from './rail-count.js?v=1'
import { mountHero } from './home-hero.js?v=1'

export function visibleVideoContinuations(items, isLibraryVisible, limit = 4) {
  return (Array.isArray(items) ? items : [])
    .filter(item => isLibraryVisible(item.library_id) && (item.stream_url || item.hls_url)
      && item.progress && !item.progress.finished && !item.progress.dismissed)
    .slice(0, Math.max(0, limit))
}

/** Owns the bounded Continue watching rail, dismissal, and undo lifecycle. */
export function createVideoContinueRail({ $, requestJSON, escapeHTML, isLibraryVisible, playerController, addArtwork, hero }) {
  let undoTimer

  async function load() {
    try {
      const result = await requestJSON('/api/video-progress')
      const active = visibleVideoContinuations(result.data, isLibraryVisible)
      // The hero shows the newest item, so the row holds the rest: one title must not appear twice.
      const row = active.slice(1)
      const rail = $('#video-reading-rail')
      rail.hidden = row.length === 0
      setRailCount(globalThis.document, 'video-reading-count', row.length)
      // Home's hero is the newest thing still in progress, and it comes from the same list as this rail.
      hero?.render(active)
      $('#video-reading-track').innerHTML = row.map(item => {
        const fraction = item.progress.duration_seconds > 0
          ? Math.min(100, item.progress.position_seconds / item.progress.duration_seconds * 100)
          : 0
        return `<article class="video-reading-card"><button class="video-reading-item" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><span class="video-art" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m9 7 8 5-8 5V7Z"/></svg></span><span><strong>${escapeHTML(cleanVideoTitle(item.title))}</strong><small>${watchedLabel(item.progress)}</small><i style="--progress:${fraction}%"></i></span></button><button class="video-reading-dismiss" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}" aria-label="Remove ${escapeHTML(item.title)} from Continue watching"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17"/></svg></button></article>`
      }).join('')
      const track = $('#video-reading-track')
      track.querySelectorAll('.video-art').forEach((art, index) => addArtwork(art, row[index]?.artwork_url))
      track.querySelectorAll('.video-reading-item').forEach(button => {
        const item = active.find(candidate => candidate.id === button.dataset.item && candidate.library_id === button.dataset.library)
        button.onclick = () => playerController.open(item, item.title)
      })
      track.querySelectorAll('.video-reading-dismiss').forEach(button => { button.onclick = async () => {
        const item = active.find(candidate => candidate.id === button.dataset.item && candidate.library_id === button.dataset.library)
        if (!item) return
        button.disabled = true
        const saveDismissed = dismissed => requestJSON(`/api/video-progress/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}`, {
          method:'PATCH', headers:{'Content-Type':'application/json'}, body:JSON.stringify({ dismissed }),
        })
        try {
          await saveDismissed(true)
          await load()
          clearTimeout(undoTimer)
          const toast = $('#undo-toast')
          $('#undo-message').textContent = `Removed “${item.title}”`
          $('#undo-action').onclick = async () => { await saveDismissed(false); toast.hidden = true; await load() }
          toast.hidden = false
          undoTimer = setTimeout(() => { toast.hidden = true }, 8000)
        } catch { button.disabled = false }
      } })
    } catch { $('#video-reading-rail').hidden = true }
  }

  return { load }
}
