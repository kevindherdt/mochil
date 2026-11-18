// Music-only filtering stays on the cold Music path; spoken-media templates do not.
export function favoriteTrackView(items, {libraryID, query, sort, natural}) {
  const needle = query.toLocaleLowerCase()
  const filtered = items.filter(item => item.library_id === libraryID && (!needle ||
    [item.title, item.artist, item.album, item.album_artist].some(value => value?.toLocaleLowerCase().includes(needle))))
  return filtered.slice().sort((a, b) => {
    if (sort === 'recent' || sort === 'oldest') {
      const result = new Date(b.modified) - new Date(a.modified)
      if (result) return sort === 'oldest' ? -result : result
    }
    if (sort === 'artist') {
      const result = natural.compare(`${a.album_artist || a.artist || ''}\0${a.album || ''}`, `${b.album_artist || b.artist || ''}\0${b.album || ''}`)
      if (result) return result
    }
    return natural.compare(a.title, b.title)
  })
}
