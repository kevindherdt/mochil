const nextOffset = response => {
  const raw = response.headers.get('X-Next-Offset')
  if (raw == null || raw === '') return null
  const value = Number(raw)
  return Number.isInteger(value) && value >= 0 ? value : null
}

/**
 * Owns the request and hierarchy state for Artist → Album → Track browsing.
 * Playback and DOM rendering deliberately stay with the audio coordinator.
 */
export function createMusicGroupCatalogue({ requestJSON }) {
  let request = null
  let navigationRequest = null
  let groups = []
  let total = 0
  let offset = null
  let selectedGroup = null
  let selectedArtist = null
  let navigation = []
  let loading = false
  let returnCount = 0 // how many artists were loaded when one was opened, so going back shows the same stretch of the list

  function abort() {
    request?.abort()
    navigationRequest?.abort()
    request = null
    navigationRequest = null
    loading = false
  }

  function reset() {
    abort()
    groups = []
    total = 0
    offset = null
    selectedGroup = null
    selectedArtist = null
    navigation = []
  }

  function clearNavigation() {
    navigationRequest?.abort()
    navigationRequest = null
    navigation = []
  }

  async function loadGroups({ library, browse, search = '', reset: replace = true, requestedOffset = 0 }) {
    if (!['albums', 'artists'].includes(browse) || selectedGroup || loading && !replace) return false
    if (replace) {
      request?.abort()
      groups = []
      total = 0
      offset = null
    }
    const controller = new AbortController()
    request = controller
    loading = true
    const start = replace ? Math.max(0, requestedOffset) : offset
    try {
      const query = new URLSearchParams({ library, group_type: browse, limit: 72, offset: start || 0 })
      if (search) query.set('q', search)
      const result = await requestJSON(`/api/audio-groups?${query}`, { signal: controller.signal })
      if (request !== controller) return false
      groups.push(...result.data)
      total = Number(result.response.headers.get('X-Total-Count')) || groups.length
      offset = nextOffset(result.response)
      return true
    } finally {
      if (request === controller) {
        request = null
        loading = false
      }
    }
  }

  async function open(group, { library, browse }) {
    request?.abort()
    const controller = new AbortController()
    request = controller
    loading = true
    try {
      if (browse === 'artists' && !selectedArtist) {
        const query = new URLSearchParams({ library, artist: group.name })
        const result = await requestJSON(`/api/audio-artist-albums?${query}`, { signal: controller.signal })
        if (request !== controller) return null
        returnCount = groups.length
        selectedArtist = group
        selectedGroup = null
        groups = result.data
        total = result.data.length
        offset = null
        return { level: 'artist', tracks: [] }
      }
      const query = new URLSearchParams({ library, group_type: 'albums', group: group.name })
      if (group.id) query.set('group_id', group.id)
      if (selectedArtist) query.set('artist', selectedArtist.name)
      const result = await requestJSON(`/api/audio-group-tracks?${query}`, { signal: controller.signal })
      if (request !== controller) return null
      selectedGroup = group
      return { level: 'album', tracks: result.data }
    } finally {
      if (request === controller) {
        request = null
        loading = false
      }
    }
  }

  async function back(context) {
    if (selectedGroup) {
      selectedGroup = null
      return 'artist-or-albums'
    }
    if (!selectedArtist) return 'root'
    selectedArtist = null
    groups = []
    total = 0
    offset = null
    await loadGroups({ ...context, browse: 'artists', reset: true })
    while (groups.length < returnCount && offset !== null && await loadGroups({ ...context, browse: 'artists', reset: false }));
    returnCount = 0
    return 'artists'
  }

  async function loadNavigation({ library, sort, mode, search = '' }) {
    navigationRequest?.abort()
    if (!['tracks', 'albums', 'artists'].includes(mode)) {
      navigation = []
      return false
    }
    const controller = new AbortController()
    navigationRequest = controller
    const query = new URLSearchParams({ library, sort, mode })
    if (search) query.set('q', search)
    try {
      const result = await requestJSON(`/api/audio-navigation?${query}`, { signal: controller.signal })
      if (navigationRequest !== controller) return false
      navigation = result.data
      return true
    } finally {
      if (navigationRequest === controller) navigationRequest = null
    }
  }

  return {
    abort,
    back,
    clearNavigation,
    findGroup: name => groups.find(group => group.name === name),
    loadGroups,
    loadNavigation,
    open,
    reset,
    get groups() { return groups },
    get loading() { return loading },
    get navigation() { return navigation },
    get nextOffset() { return offset },
    get selectedArtist() { return selectedArtist },
    get selectedGroup() { return selectedGroup },
    get total() { return total },
  }
}
