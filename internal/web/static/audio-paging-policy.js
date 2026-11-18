const staticMusicDestinations = new Set([
  'discover',
  'recently-played',
  'favorites',
  'playlists',
  'queue',
])

/**
 * Keeps paging rules independent from DOM observers and catalogue rendering.
 * Each catalogue still owns its request state; this policy only decides whether
 * another page is meaningful and which loader should receive the request.
 */
export function audioPagingDecision({
  visible = true,
  kind = '',
  browse = '',
  loading = false,
  itemCount = 0,
  total = 0,
  startOffset = 0,
  podcast = {},
  audiobook = {},
  music = {},
} = {}) {
  const groupedAudiobooks = kind === 'audiobooks' && ['series', 'authors'].includes(browse)
  const groupedMusic = kind === 'music' && ['albums', 'artists'].includes(browse)
  const incrementalMusic = kind === 'music' && ['recently-played', 'favorites'].includes(browse)
  let canLoad = false
  let target = 'audio-items'

  if (visible && kind === 'podcasts') {
    target = 'podcast-show'
    canLoad = Boolean(podcast.show) && podcast.count < podcast.total && !podcast.loading
  } else if (visible && groupedAudiobooks) {
    target = 'audiobook-group'
    canLoad = Boolean(audiobook.group) && audiobook.count < audiobook.total && !audiobook.loading
  } else if (visible && incrementalMusic && music.localHasMore) {
    target = 'local-music'
    canLoad = true
  } else if (visible && groupedMusic) {
    target = 'music-groups'
    canLoad = !music.selectedGroup && music.count < music.total && music.nextOffset !== null
  } else if (visible) {
    const loaded = kind === 'music' && browse === 'tracks' ? startOffset + itemCount : itemCount
    canLoad = !(kind === 'music' && staticMusicDestinations.has(browse)) && loaded < total && !loading
  }

  return {
    canLoad,
    target,
    autoLoad:
      kind === 'music' && ['tracks', 'albums', 'artists'].includes(browse) ||
      incrementalMusic && music.localHasMore ||
      kind === 'audiobooks' && !groupedAudiobooks,
  }
}

export function canLoadPreviousMusicPage({ visible = true, kind = '', browse = '', loading = false, startOffset = 0 } = {}) {
  return visible && kind === 'music' && browse === 'tracks' && !loading && startOffset > 0
}
