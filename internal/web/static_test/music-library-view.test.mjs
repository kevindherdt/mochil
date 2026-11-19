import assert from 'node:assert/strict'
import test from 'node:test'
import { musicAccountLoadStatus, musicAlbumTrackColumns, musicLibraryView } from '../static/music-library-view.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('"', '&quot;')
const row = item => `<article>${escapeHTML(item.title)}</article>`
const card = item => `<button>${escapeHTML(item.title)}</button>`
const base = {
  discovery: { played: [], recent: [], favorites: [], albums: [], artists: [] },
  catalogue: { groups: [], total: 0, selectedGroup: null, selectedArtist: null },
  playlists: [], selectedPlaylist: '', favorites: [], queue: [], items: [], layout: 'list', row, card,
  albumRow: row, escapeHTML,
}

test('music destinations keep a stable status-free root and purpose-owned empty states', () => {
  assert.equal(musicLibraryView({ ...base, browse: 'discover' }).status, '')
  assert.equal(musicLibraryView({ ...base, browse: 'recently-played' }).status, '')
  assert.equal(musicLibraryView({ ...base, browse: 'favorites' }).status, '')
  assert.match(musicLibraryView({ ...base, browse: 'favorites' }).html, /Tracks you favorite will appear here/)
  assert.match(musicLibraryView({ ...base, browse: 'queue' }).html, /Add tracks to build your queue/)
  assert.equal(musicLibraryView({ ...base, browse: 'tracks' }), null)
})

test('music account failures are not presented as genuinely empty collections', () => {
  const loadErrors = { favorites: 'request failed', playlists: 'request failed' }
  assert.equal(musicAccountLoadStatus(loadErrors), 'Some music account data could not load: favorites and playlists.')
  assert.match(musicLibraryView({ ...base, browse: 'favorites', loadErrors }).status, /Could not load favorites/)
  assert.match(musicLibraryView({ ...base, browse: 'playlists', loadErrors }).status, /Could not load playlists/)
  assert.match(musicLibraryView({ ...base, browse: 'discover', loadErrors }).status, /favorites and playlists/)
})

test('recently played keeps history but omits resumable track progress', () => {
  const calls = []
  const view = musicLibraryView({
    ...base,
    browse: 'recently-played',
    discovery: { ...base.discovery, played: [{ title: 'Song' }] },
    row: (item, dashboard, showProgress) => {
      calls.push([item.title, dashboard, showProgress])
      return row(item)
    },
  })
  assert.match(view.html, /Song/)
  assert.deepEqual(calls, [['Song', false, false]])
})

test('large in-memory music collections render an incremental first window', () => {
  const favorites = Array.from({ length: 250 }, (_, index) => ({ title: `Song ${index}` }))
  const view = musicLibraryView({ ...base, browse: 'favorites', favorites, collectionLimit: 120 })
  assert.equal((view.html.match(/<article>/g) || []).length, 120)
  assert.equal(view.hasMore, true)
  const complete = musicLibraryView({ ...base, browse: 'favorites', favorites, collectionLimit: 300 })
  assert.equal((complete.html.match(/<article>/g) || []).length, 250)
  assert.equal(complete.hasMore, false)
})

test('music playlist view clears a deleted selection without losing the overview', () => {
  const view = musicLibraryView({ ...base, browse: 'playlists', selectedPlaylist: 'Removed', playlists: [{ name: 'Kept', items: [] }] })
  assert.equal(view.selectedPlaylist, '')
  assert.equal(view.status, '')
  assert.match(view.html, /data-open-playlist="Kept"/)
  assert.match(view.html, /music-playlist-grid list/)
})

test('detail surfaces own their identity without repeating catalogue status', () => {
  const dashboardFlags = []
  const queue = musicLibraryView({
    ...base,
    browse: 'queue',
    queue: [{ title: 'First' }, { title: 'Second' }],
    row: (item, dashboard = false) => {
      dashboardFlags.push(dashboard)
      return row(item)
    },
  })
  assert.equal(queue.status, '')
  assert.match(queue.html, />2 tracks</)
  assert.deepEqual(dashboardFlags, [false, false])
  const album = { name: 'Album & One', count: 1, subtitle: 'Artist' }
  const focused = musicLibraryView({ ...base, browse: 'albums', items: [{ title: 'Song' }], catalogue: { ...base.catalogue, selectedGroup: album } })
  assert.equal(focused.status, '')
  assert.match(focused.html, /Album &amp; One/)
  assert.match(focused.html, /class="music-album-tracks"/)

  const playlist = musicLibraryView({
    ...base,
    browse: 'playlists',
    selectedPlaylist: 'Evening',
    playlists: [{ name: 'Evening', items: [{ title: 'Song' }] }],
  })
  assert.equal(playlist.status, '')
  assert.match(playlist.html, /Evening/)
})

test('wide album columns preserve sequential track order', () => {
  const indexes = []
  const html = musicAlbumTrackColumns([1, 2, 3, 4, 5], (value, index) => { indexes.push(index); return `<i>${value}</i>` })
  assert.match(html, /music-album-track-column"><i>1<\/i><i>2<\/i><i>3<\/i><\/div><div class="music-album-track-column"><i>4<\/i><i>5<\/i>/)
  assert.deepEqual(indexes, [0, 1, 2, 3, 4])
})
