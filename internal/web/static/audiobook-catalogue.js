/** Owns the audiobook series → books hierarchy, paging, and stale requests. */
export function createAudiobookCatalogue({ requestJSON, pageSize = 120 }) {
  let request = null
  let selectedGroup = ''
  let series = []
  let books = []
  let total = 0
  let nextCursor = ''
  let loading = false
  let groupType = 'series'

  function abort() {
    request?.abort()
    request = null
    loading = false
  }

  function reset() {
    abort()
    selectedGroup = ''
    series = []
    books = []
    total = 0
    nextCursor = ''
    groupType = 'series'
  }

  async function loadAuthors({ library, search = '' }) {
    abort()
    selectedGroup = ''
    books = []
    total = 0
    nextCursor = ''
    groupType = 'authors'
    const current = new AbortController()
    request = current
    loading = true
    const query = new URLSearchParams({ library, group_type: 'authors', limit: 120 })
    if (search) query.set('q', search)
    try {
      const result = await requestJSON(`/api/audio-groups?${query}`, { signal: current.signal })
      if (request !== current) return false
      series = (result.data || []).map(group => ({
        ...group,
        artwork_url: group.artwork_item_id
          ? `/api/audio-artwork/${encodeURIComponent(library)}/${encodeURIComponent(group.artwork_item_id)}?size=card`
          : '',
      }))
      return true
    } finally {
      if (request === current) {
        request = null
        loading = false
      }
    }
  }

  async function loadSeries({ library, search = '', sort = 'series' }) {
    abort()
    selectedGroup = ''
    books = []
    total = 0
    nextCursor = ''
    groupType = 'series'
    const current = new AbortController()
    request = current
    loading = true
    const query = new URLSearchParams({ library, sort })
    if (search) query.set('q', search)
    try {
      const result = await requestJSON(`/api/audiobook-series?${query}`, { signal: current.signal })
      if (request !== current) return false
      series = result.data || []
      return true
    } finally {
      if (request === current) {
        request = null
        loading = false
      }
    }
  }

  async function loadGroup(name, { library, search = '', reset: restart = true } = {}) {
    if (loading && !restart) return false
    selectedGroup = name
    if (restart) {
      abort()
      books = []
      total = 0
      nextCursor = ''
    }
    const current = new AbortController()
    request = current
    loading = true
    const selected = series.find(group => group.name === name)
    const query = groupType === 'authors'
      ? new URLSearchParams({ library, group_type: 'authors', group: name, group_id: selected?.id || '' })
      : new URLSearchParams({ library, group: name, limit: pageSize, sort: 'series' })
    if (books.length) query.set(nextCursor ? 'cursor' : 'offset', nextCursor || books.length)
    if (search) query.set('q', search)
    try {
      const endpoint = groupType === 'authors' ? '/api/audio-group-tracks' : '/api/audio'
      const result = await requestJSON(`${endpoint}?${query}`, { signal: current.signal })
      if (request !== current) return false
      const received = result.data || []
      total = Number(result.response.headers.get('X-Total-Count')) || (books.length + received.length)
      nextCursor = result.response.headers.get('X-Next-Cursor') || ''
      books.push(...received)
      return true
    } catch (error) {
      if (error.status === 409 && !restart) {
        if (request === current) {
          request = null
          loading = false
        }
        return loadGroup(name, { library, search, reset: true })
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
    loadSeries,
    loadAuthors,
    loadGroup,
    get group() { return selectedGroup },
    get series() { return series },
    get books() { return books },
    get total() { return total },
    get loading() { return loading },
    get groupType() { return groupType },
  }
}
