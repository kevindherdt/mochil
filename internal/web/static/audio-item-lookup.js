// Which list a clicked row belongs to. The music "browse" menu only means something in the music library: its last
// value lingers when you move to podcasts or audiobooks, and must not decide where their rows are looked up.
export function lookupPool({ kind, browse, discovery, playlistItems, visibleItems }) {
  if (kind === 'music' && browse === 'discover') return [...discovery.played, ...discovery.recent, ...discovery.favorites]
  if (kind === 'music' && browse === 'playlists') return playlistItems
  return visibleItems()
}
