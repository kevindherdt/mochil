/** Owns the selected podcast show, episode paging, and stale requests. */
export function createPodcastCatalogue({ requestJSON, pageSize = 500 }) {
  let request = null
  let selectedShow = ''
  let episodes = []
  let total = 0
  let nextCursor = ''
  let loading = false

  function abort() {
    request?.abort()
    request = null
    loading = false
  }

  function reset() {
    abort()
    selectedShow = ''
    episodes = []
    total = 0
    nextCursor = ''
  }

  function closeShow() {
    abort()
    selectedShow = ''
    episodes = []
    total = 0
    nextCursor = ''
  }

  async function loadShow(name, {
    library,
    search = '',
    sort = 'recent',
    played = 'all',
    reset: restart = true,
  } = {}) {
    if (loading && !restart) return false
    selectedShow = name
    if (restart) {
      abort()
      episodes = []
      total = 0
      nextCursor = ''
    }
    const current = new AbortController()
    request = current
    loading = true
    const query = new URLSearchParams({ library, show: name, limit: pageSize, sort, played })
    if (episodes.length) query.set(nextCursor ? 'cursor' : 'offset', nextCursor || episodes.length)
    if (search) query.set('q', search)
    try {
      const result = await requestJSON(`/api/audio?${query}`, { signal: current.signal })
      if (request !== current) return false
      const received = result.data || []
      total = Number(result.response.headers.get('X-Total-Count')) || (episodes.length + received.length)
      nextCursor = result.response.headers.get('X-Next-Cursor') || ''
      episodes.push(...received)
      return true
    } catch (error) {
      if (error.status === 409 && !restart) {
        if (request === current) {
          request = null
          loading = false
        }
        return loadShow(name, { library, search, sort, played, reset: true })
      }
      throw error
    } finally {
      if (request === current) {
        request = null
        loading = false
      }
    }
  }

  return {
    abort,
    reset,
    closeShow,
    loadShow,
    get show() { return selectedShow },
    get episodes() { return episodes },
    get total() { return total },
    get loading() { return loading },
  }
}
