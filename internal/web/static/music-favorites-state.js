/** Owns favorites reads, per-track writes, and reconciliation; no queue/playlist state. */
export function createMusicFavoritesState({requestJSON, key, onChanged}) {
  let items = [], error = '', version = 0
  const keys = new Set(), writes = new Map(), reads = new Set()

  async function load() {
    const current = ++version, changes = new Map()
    reads.add(changes)
    try {
      const loaded = (await requestJSON('/api/audio-favorites')).data
      if (current !== version) return
      if (loaded !== null && !Array.isArray(loaded)) throw new Error('Invalid favorites response')
      const merged = new Map((loaded || []).map(item => [key(item), item]))
      for (const [itemKey, item] of changes) {
        if (item) merged.set(itemKey, item)
        else merged.delete(itemKey)
      }
      items = [...merged.values()]
      keys.clear()
      items.forEach(item => keys.add(key(item)))
      error = ''
    } catch (failure) {
      if (current === version) error = failure?.message || 'Favorites could not be loaded.'
    } finally { reads.delete(changes) }
  }

  async function toggle(item) {
    if (!item) return false
    // Capture the selected track before waiting behind another toggle.
    const selected = {...item}
    const itemKey = key(selected)
    const previous = writes.get(itemKey) || Promise.resolve()
    const write = previous.catch(() => {}).then(async () => {
      const favorite = !keys.has(itemKey)
      await requestJSON('/api/audio-favorites', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({library_id:selected.library_id,item_id:selected.id,favorite})})
      for (const changes of reads) changes.set(itemKey, favorite ? selected : null)
      if (favorite) {
        keys.add(itemKey)
        if (!items.some(candidate => key(candidate) === itemKey)) items = [...items, selected]
      } else {
        keys.delete(itemKey)
        items = items.filter(candidate => key(candidate) !== itemKey)
      }
      onChanged()
      return favorite
    })
    writes.set(itemKey, write)
    try { return await write }
    finally { if (writes.get(itemKey) === write) writes.delete(itemKey) }
  }

  return {load, toggle, isFavorite:item => keys.has(key(item)),
    get items() { return items }, get error() { return error }}
}
