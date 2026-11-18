export function createDestinationScrollMemory({ viewport = globalThis, maximum = 24 } = {}) {
  const positions = new Map()

  function remember(key) {
    if (!key) return
    positions.delete(key)
    positions.set(key, Math.max(0, Number(viewport.scrollY) || 0))
    while (positions.size > maximum) positions.delete(positions.keys().next().value)
  }

  function restore(key) {
    const top = positions.get(key)
    if (top === undefined) return false
    const apply = () => viewport.scrollTo?.({ top, behavior: 'auto' })
    if (typeof viewport.requestAnimationFrame === 'function') {
      viewport.requestAnimationFrame(() => viewport.requestAnimationFrame(apply))
    } else apply()
    return true
  }

  return { remember, restore }
}

export function createMusicDestinationState({ select, sort, storage, viewport, getLibrary, closeMenu, resetWindow, resetPlaylist, hideCreate, reload }) {
  const memory = createDestinationScrollMemory({ viewport })
  let previous = select.value
  const key = browse => `${getLibrary()}\0${browse}`
  return {
    setCurrent(value) { previous = value },
    select(value) {
      select.value = value
      previous = value
      storage.setItem('bobobooks-music-browse', value)
    },
    selectSort(value) {
      sort.value = value
      storage.setItem('bobobooks-audio-sort-music', value)
    },
    async change() {
      closeMenu()
      memory.remember(key(previous))
      resetWindow()
      const browse = select.value
      previous = browse
      if (browse !== 'playlists') resetPlaylist()
      storage.setItem('bobobooks-music-browse', browse)
      hideCreate()
      await reload()
      memory.restore(key(browse))
    },
  }
}

export function restoreAudioDestination({ kind, sort, savedSort, music, audiobooks, storage, onMusic }) {
  const fallback = kind === 'music' ? 'title' : kind === 'audiobooks' ? 'series' : 'recent'
  sort.value = [...sort.options].some(option => option.value === savedSort) ? savedSort : fallback
  if (kind === 'music') {
    music.value = storage.getItem('bobobooks-music-browse') || 'discover'
    onMusic(music.value)
  }
  if (kind === 'audiobooks') audiobooks.value = storage.getItem('bobobooks-audiobook-browse') || 'series'
}
