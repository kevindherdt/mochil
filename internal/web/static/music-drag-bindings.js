import {createMusicOrderDrag} from './music-order-drag.js?v=1'
import {playlistItemsAfterDrop} from './music-state.js?v=14'

/** Connect drag gestures to playlist intent and optimistic queue state. */
export function createMusicDragBindings({container, music, selectedPlaylist, key, render, status}) {
  const playlistDrag = createMusicOrderDrag({
    container, rowSelector:'.music-playlist-tracks > .audio-item',
    onMove:async (from, target, after) => {
      const list = music.playlists.find(value => value.name === selectedPlaylist())
      if (!list?.items?.[from] || !list.items[target]) return
      const movedKey = key(list.items[from]), targetKey = key(list.items[target])
      if (movedKey === targetKey) return
      try {
        await music.editPlaylist(list.name, items => playlistItemsAfterDrop(
          items, items.findIndex(item => key(item) === movedKey),
          items.findIndex(item => key(item) === targetKey), after,
        ))
        status(`${list.name} order saved.`)
      } catch (error) { status(`Could not reorder this playlist: ${error.message}`) }
    },
  })
  const queueDrag = createMusicOrderDrag({
    container, rowSelector:'.queue-group > .audio-item',
    onMove:async (from, target, after) => {
      const reordered = playlistItemsAfterDrop(music.queue, from, target, after)
      if (reordered.every((item, index) => key(item) === key(music.queue[index]))) return
      music.setQueue(reordered)
      render()
      status('Queue reordered.')
    },
  })
  return {playlistDrag, queueDrag}
}
