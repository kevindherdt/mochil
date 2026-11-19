export function visibleVideoContinuations(items, isLibraryVisible, limit = 4) {
  return (Array.isArray(items) ? items : [])
    .filter(item => isLibraryVisible(item.library_id) && (item.stream_url || item.hls_url)
      && item.progress && !item.progress.finished && !item.progress.dismissed)
    .slice(0, Math.max(0, limit))
}

/** Owns the bounded Continue watching rail, dismissal, and undo lifecycle. */
export function createVideoContinueRail({ $, requestJSON, escapeHTML, isLibraryVisible, playerController, addArtwork }) {
  let undoTimer

  async function load() {
    try {
      const result = await requestJSON('/api/video-progress')
      const active = visibleVideoContinuations(result.data, isLibraryVisible)
      const rail = $('#video-reading-rail')
      rail.hidden = active.length === 0
      $('#video-reading-track').innerHTML = active.map(item => {
        const fraction = item.progress.duration_seconds > 0
          ? Math.min(100, item.progress.position_seconds / item.progress.duration_seconds * 100)
          : 0
        return `<article class="video-reading-card"><button class="video-reading-item" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}"><span class="video-art" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m9 7 8 5-8 5V7Z"/></svg></span><span><strong>${escapeHTML(item.title)}</strong><small>${Math.floor(item.progress.position_seconds / 60)} min watched</small><i style="--progress:${fraction}%"></i></span></button><button class="video-reading-dismiss" data-library="${escapeHTML(item.library_id)}" data-item="${escapeHTML(item.id)}" aria-label="Remove ${escapeHTML(item.title)} from Continue watching"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M17 7 7 17"/></svg></button></article>`
      }).join('')
      const track = $('#video-reading-track')
      track.querySelectorAll('.video-art').forEach((art, index) => addArtwork(art, active[index]?.artwork_url))
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
