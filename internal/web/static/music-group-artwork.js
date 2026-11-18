export function musicGroupArtworkURL(group = {}) {
  return group.artwork_item_id && group.artwork_library_id
    ? `/api/audio-artwork/${encodeURIComponent(group.artwork_library_id)}/${encodeURIComponent(group.artwork_item_id)}`
    : ''
}

/**
 * Give focused album tracks presentation artwork without adding repeated covers
 * to the track list. Existing track-specific artwork always wins.
 */
export function musicGroupPlaybackTracks(tracks, group) {
  if (!Array.isArray(tracks)) return []
  const artwork = musicGroupArtworkURL(group)
  if (!artwork) return tracks
  return tracks.map(track => track.player_artwork_url || track.artwork_url
    ? track
    : { ...track, player_artwork_url: artwork })
}
