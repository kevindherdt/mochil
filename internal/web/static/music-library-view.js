import { musicDiscoveryHTML } from './music-discovery.js?v=10'
import { installMusicNames } from './music-names.js?v=5'
installMusicNames()
import { musicGroupArtworkURL, musicGroupGridHTML, musicTrackGroupHeadingHTML } from './music-group-view.js?v=7'
import { browseHeadingHTML, musicPlaylistDetailHTML, musicPlaylistOverviewHTML } from './music-playlist-view.js?v=8'
import { musicTrackCollectionHTML } from './music-track-collection.js?v=1'

/** Pure composition for every bounded music destination except the virtualized Tracks feed. */
export function musicLibraryView({ browse, discovery, catalogue, playlists, selectedPlaylist, favorites, queue, items, layout, row, card, albumRow, escapeHTML, loadErrors = {}, collectionLimit = Number.POSITIVE_INFINITY }) {
  if (browse === 'discover') return {
    status: musicAccountLoadStatus(loadErrors),
    html: musicDiscoveryHTML({ ...discovery, playlists }, { row, card, escapeHTML, groupArtworkURL: musicGroupArtworkURL }),
    selectedPlaylist,
  }
  if (browse === 'recently-played') {
    const list = discovery.played
    const visible = list.slice(0, collectionLimit)
    return {
      status: '',
      html: browseHeadingHTML('Recently played', list.length, 'discover', 'Back to Discover') + musicTrackCollectionHTML(visible, {
        layout,
        row: item => row(item, false, false),
        card,
        empty: 'Music you play will appear here.',
      }),
      selectedPlaylist,
      hasMore: visible.length < list.length,
    }
  }
  if (['albums', 'artists'].includes(browse) && catalogue.selectedGroup) {
    const group = catalogue.selectedGroup
    return {
      status: '',
      html: musicTrackGroupHeadingHTML(group, catalogue.selectedArtist?.name || group.subtitle, escapeHTML) + musicAlbumTrackColumns(items, albumRow),
      selectedPlaylist,
    }
  }
  if (['albums', 'artists'].includes(browse)) {
    const artist = catalogue.selectedArtist?.name || ''
    return {
      status: '',
      html: musicGroupGridHTML(catalogue.groups, { escapeHTML, type: catalogue.selectedArtist ? 'albums' : browse, title: artist, backLabel: catalogue.selectedArtist ? 'Back to artists' : '', layout }),
      selectedPlaylist,
    }
  }
  if (browse === 'playlists') {
    const playlist = playlists.find(value => value.name === selectedPlaylist)
    const current = playlist ? selectedPlaylist : ''
    return {
      status: loadErrors.playlists ? `Could not load playlists: ${loadErrors.playlists}` : '',
      html: playlist ? musicPlaylistDetailHTML(playlist, row, escapeHTML) : musicPlaylistOverviewHTML(playlists, escapeHTML, layout, favorites.length),
      selectedPlaylist: current,
    }
  }
  if (browse === 'favorites') return {
    status: loadErrors.favorites ? `Could not load favorites: ${loadErrors.favorites}` : '',
    html: browseHeadingHTML('Favorites', favorites.length, 'playlists', 'Back to playlists') + musicTrackCollectionHTML(favorites.slice(0, collectionLimit), { layout, row, card, empty: 'Tracks you favorite will appear here.' }),
    selectedPlaylist,
    hasMore: favorites.length > collectionLimit,
  }
  if (browse === 'queue') return {
    status: loadErrors.queue || '',
    html: (loadErrors.queue ? '<button data-retry-queue-save>Retry queue sync</button>' : '') + (queue.length
      ? `<section class="audio-group playlist-group queue-group"><header><h2>Queue</h2><span>${queue.length.toLocaleString()} ${queue.length === 1 ? 'track' : 'tracks'}</span><button data-clear-queue>Clear</button></header>${queue.map(item => row(item)).join('')}</section>`
      : '<p class="empty-state">Add tracks to build your queue.</p>'),
    selectedPlaylist,
  }
  return null
}

export function musicAccountLoadStatus(loadErrors = {}) {
  const failed = [loadErrors.favorites && 'favorites', loadErrors.playlists && 'playlists'].filter(Boolean)
  return failed.length ? `Some music account data could not load: ${failed.join(' and ')}.` : ''
}

export function musicAlbumTrackColumns(items, row) {
  const midpoint = Math.ceil(items.length / 2)
  const column = (values, offset) => `<div class="music-album-track-column">${values.map((item, index) => row(item, index + offset)).join('')}</div>`
  return `<div class="music-album-tracks">${column(items.slice(0, midpoint), 0)}${column(items.slice(midpoint), midpoint)}</div>`
}
