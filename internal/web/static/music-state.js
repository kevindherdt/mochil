import { createMusicQueueState } from './music-queue-state.js?v=1'
export { queueWithNext } from './music-queue-state.js?v=1'
import { decodePlaylistOverview, decodePlaylistDetail } from './music-playlist-data.js?v=1'
import { createMusicFavoritesState } from './music-favorites-state.js?v=1'

export function createMusicState({ requestJSON, key, onChanged }) {
  let playlists = []
  let loadErrors = { favorites: '', playlists: '' }
  let libraryLoaded = false, libraryLoading
  const queueState = createMusicQueueState({requestJSON, key, onChanged})
  const favoriteState = createMusicFavoritesState({requestJSON, key, onChanged})
  let playlistWrite = Promise.resolve()
  let playlistVersion = 0
  const playlistReads = new Map()
  const asItems = value => Array.isArray(value) ? value : []
  const entries = items => asItems(items).map(item => ({library_id:item.library_id, item_id:item.id}))
  async function loadPlaylistsWhenIdle() {
    // A refresh started during a write must not race its completion. Recheck the
    // queue tail because another edit can be queued while this one is awaited.
    while (true) {
      const pending = playlistWrite
      await pending.catch(() => {})
      if (pending === playlistWrite) return loadPlaylists()
    }
  }
  async function loadPlaylists() {
    const version = ++playlistVersion
    try {
      const loaded = decodePlaylistOverview((await requestJSON('/api/audio-playlists?view=summary')).data)
      if (version !== playlistVersion) return
      playlists = loaded
      loadErrors = { ...loadErrors, playlists: '' }
    } catch (error) {
      if (version !== playlistVersion) return
      loadErrors = { ...loadErrors, playlists: error?.message || 'Playlists could not be loaded.' }
    }
  }
  async function ensurePlaylist(name) {
    const current = playlists.find(value=>value.name===name)
    if (!current || current.detailLoaded !== false) return current
    const version = playlistVersion
    const pending = playlistReads.get(name)
    if (pending?.version === version) return pending.promise
    const promise = loadPlaylistDetail(name, version).finally(() => {
      if (playlistReads.get(name)?.promise === promise) playlistReads.delete(name)
    })
    playlistReads.set(name, {version, promise})
    return promise
  }
  async function loadPlaylistDetail(name, version) {
    try {
      const result = await requestJSON(`/api/audio-playlists/${encodeURIComponent(name)}`)
      const detail = {...decodePlaylistDetail(result.data, name), revision:result.response?.headers?.get('ETag') || undefined}
      if (result.response?.headers?.get('X-Mochil-Playlist-Edit') === 'visible-v1') detail.visibleEdit = true
      if (version !== playlistVersion) return
      playlists = playlists.map(value=>value.name===name?detail:value)
      return detail
    } catch(error) {
      if (version !== playlistVersion) return
      playlists = playlists.map(value=>value.name===name?{...value,detailError:error.message}:value)
    }
  }
  async function load({force=false}={}) {
    if (libraryLoading && !force) { await libraryLoading; return }
    if (libraryLoaded && !force) return
    const current = Promise.all([favoriteState.load(), loadPlaylistsWhenIdle()]).then(() => {
      if (libraryLoading === current) libraryLoaded = true
    }).finally(() => {
      if (libraryLoading === current) libraryLoading = null
    })
    libraryLoading = current
    await current
  }
  function mutatePlaylist(action) {
    const write = playlistWrite.catch(() => {}).then(() => { playlistVersion++; return action() })
    playlistWrite = write
    return write
  }
  function savePlaylist(name, items) {
    const normalized = asItems(items).map(item => ({...item}))
    return mutatePlaylist(() => writePlaylist(name, normalized))
  }
  async function conditionalPlaylistRequest(name, options, suffix='') {
    const current = await ensurePlaylist(name)
    if (!current || current.detailLoaded === false) throw new Error('Could not load this playlist. No changes were sent.')
    const headers = {...options.headers}
    if (current?.revision) headers['If-Match'] = current.revision
    try {
      return await requestJSON(`/api/audio-playlists/${encodeURIComponent(name)}${suffix}`, {...options, headers})
    } catch (error) {
      if (error?.status === 412) {
        playlists = playlists.map(list=>list.name===name?{...list,detailLoaded:false}:list)
        await loadPlaylistDetail(name, playlistVersion)
        onChanged()
      }
      throw error
    }
  }
  async function writePlaylist(name, normalized, visible) {
    const current = await ensurePlaylist(name)
    const preserve = visible !== undefined && current?.visibleEdit === true
    if (preserve && !current.revision) throw new Error('Could not verify this playlist revision. No changes were sent.')
    const result = await conditionalPlaylistRequest(name, {
      method:'PUT', headers:{'Content-Type':'application/json'},
      body:JSON.stringify(preserve ? {visible_entries:entries(visible), entries:entries(normalized)} : entries(normalized)),
    }, preserve ? '?edit=visible' : '')
    const revision = result.response?.headers?.get('ETag')
    const updated = {name, items:normalized, ...(revision ? {revision} : {}), ...(current?.visibleEdit ? {visibleEdit:true} : {})}
    const index = playlists.findIndex(playlist => playlist.name === name)
    if (index < 0) playlists = [...playlists, updated]
    else playlists = playlists.map((playlist, position) => position === index ? updated : playlist)
    onChanged()
  }
  function editPlaylist(name, transform) {
    return mutatePlaylist(async () => {
      const list = await ensurePlaylist(name)
      if (!list || list.detailLoaded === false) throw new Error('Could not load this playlist. No changes were sent.')
      // Evaluate intent after preceding writes, never against a click-time snapshot.
      const updated = transform(asItems(list.items).map(item => ({...item})))
      if (!Array.isArray(updated)) throw new Error('Invalid playlist edit.')
      await writePlaylist(name, updated, asItems(list.items))
    })
  }
  function createPlaylist(name) { return mutatePlaylist(async () => {
      await requestJSON('/api/audio-playlists', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name})})
      if (!playlists.some(playlist => playlist.name === name)) playlists = [...playlists, {name, items:[], detailLoaded:false}]
      onChanged()
    }) }
  function addPlaylistItem(name, item) {
    const body = JSON.stringify({library_id:item.library_id,item_id:item.id})
    return mutatePlaylist(async () => {
      await requestJSON(`/api/audio-playlists/${encodeURIComponent(name)}`, {method:'POST', headers:{'Content-Type':'application/json'}, body})
      playlists = playlists.map(playlist => playlist.name === name ? {...playlist, detailLoaded:false} : playlist)
      // Reconcile server state rather than claiming the old local list includes
      // additions made by another device. Never PUT a stale overview snapshot.
      await loadPlaylists()
      onChanged()
    }) }
  function renamePlaylist(name, newName) { return mutatePlaylist(async () => {
      await conditionalPlaylistRequest(name, {method:'PATCH', headers:{'Content-Type':'application/json'}, body:JSON.stringify({name:newName})})
      playlists = playlists.map(playlist => playlist.name === name ? {...playlist, name:newName, detailLoaded:false} : playlist)
      onChanged()
    }) }
  function deletePlaylist(name) { return mutatePlaylist(async () => {
      await conditionalPlaylistRequest(name, {method:'DELETE'})
      playlists = playlists.filter(playlist => playlist.name !== name)
      onChanged()
    }) }
  return {
    load, ensurePlaylist, toggleFavorite:favoriteState.toggle, savePlaylist, editPlaylist, createPlaylist, addPlaylistItem, renamePlaylist, deletePlaylist,
    ensureQueue:queueState.ensureQueue, setQueue:queueState.setQueue,
    playNext:queueState.playNext, addToQueue:queueState.addToQueue,
    removeFromQueue:queueState.removeFromQueue, moveInQueue:queueState.moveInQueue,
    clearQueue:queueState.clearQueue, retryQueueSave:queueState.retryQueueSave,
    isFavorite:favoriteState.isFavorite,
    get queue() { return queueState.items },
    get favoriteItems() { return favoriteState.items },
    get playlists() { return playlists },
    get playlistItems() { return playlists.flatMap(list => asItems(list.items)) },
    get loadErrors() {
      const errors = {...loadErrors, favorites:favoriteState.error}
      return queueState.error ? {...errors, queue:queueState.error} : errors
    },
  }
}

export function playlistItemsAfterAction(items, item, action, key) {
  const index = items.findIndex(candidate => key(candidate) === key(item))
  if (index < 0) return items.slice()
  if (action === 'playlist-remove') return items.filter((_, position) => position !== index)
  const delta = action === 'playlist-up' ? -1 : action === 'playlist-down' ? 1 : 0
  const target = index + delta
  if (!delta || target < 0 || target >= items.length) return items.slice()
  const next = items.slice()
  ;[next[index], next[target]] = [next[target], next[index]]
  return next
}

export function playlistItemsAfterDrop(items, fromIndex, targetIndex, placeAfter) {
  if (fromIndex < 0 || fromIndex >= items.length || targetIndex < 0 || targetIndex >= items.length) return items.slice()
  const insertion = targetIndex + (placeAfter ? 1 : 0)
  const next = items.slice()
  const [moved] = next.splice(fromIndex, 1)
  const adjusted = insertion > fromIndex ? insertion - 1 : insertion
  next.splice(adjusted, 0, moved)
  return next
}
