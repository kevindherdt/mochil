import { setRailCount } from './rail-count.js?v=1'

export function continuedAudioItems(values, isLibraryVisible, limit = 4) {
  return values.filter(item =>
    item.library_kind !== 'music' &&
    isLibraryVisible(item.library_id) &&
    item.progress &&
    !item.progress.finished &&
    !item.progress.dismissed,
  ).slice(0, limit)
}

export function createAudioContinueRail({ rail, track, requestJSON, isLibraryVisible, row, artwork, onOpen }) {
  let items = []
  let request

  function itemFor(target) {
    const article = target.closest('.audio-item')
    return items.find(item => item.id === article?.dataset.item && item.library_id === article?.dataset.library)
  }

  async function load() {
    request?.abort()
    const current = new AbortController()
    request = current
    try {
      const result = await requestJSON('/api/audio-progress', { signal: current.signal })
      if (request !== current) return
      items = continuedAudioItems(result.data, isLibraryVisible)
      rail.hidden = !items.length
      setRailCount(globalThis.document, 'audio-reading-count', items.length)
      track.innerHTML = items.map(item => row(item, true)).join('')
      rail.querySelectorAll('.audio-glyph').forEach(element => artwork(element, itemFor(element)?.artwork_url))
    } catch (error) {
      if (request === current && error.name !== 'AbortError') {
        items = []
        rail.hidden = true
      }
    } finally {
      if (request === current) request = undefined
    }
  }

  rail.addEventListener('click', async event => {
    const item = itemFor(event.target)
    if (!item) return
    if (event.target.closest('.audio-dismiss')) {
      await requestJSON(`/api/audio-progress/${encodeURIComponent(item.library_id)}/${encodeURIComponent(item.id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dismissed: true }),
      })
      await load()
    } else if (event.target.closest('.audio-open')) onOpen(item)
  })

  return {
    load,
    dispose() {
      request?.abort()
      request = undefined
      items = []
    },
  }
}
