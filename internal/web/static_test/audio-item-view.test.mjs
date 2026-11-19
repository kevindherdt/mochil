import assert from 'node:assert/strict'
import test from 'node:test'
import { audioClock, createAudioItemView } from '../static/audio-item-view.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')
const view = (overrides = {}) => createAudioItemView({
  escapeHTML,
  relativePublishedDate: () => '2 days ago',
  playlists: () => [],
  isFavorite: () => false,
  browse: () => 'tracks',
  ...overrides,
})

test('audio clock formats bounded media durations', () => {
  assert.equal(audioClock(0), '0:00')
  assert.equal(audioClock(125), '2:05')
  assert.equal(audioClock(7_504), '2:05:04')
  assert.equal(audioClock(Number.NaN), '0:00')
})

test('ordinary music rows expose actions without dashboard dismissal', () => {
  const html = view().row({ library_id:'music', id:'1', library_kind:'music', title:'A & B', artist:'Artist', duration_seconds:125, download_url:'/audio/music/1/A.mp3?download=1', progress:{ position_seconds:50, duration_seconds:100 } })
  assert.match(html, /A &amp; B/)
  assert.match(html, /Add to queue/)
  assert.match(html, /download/)
  assert.match(html, /Download/)
  assert.match(html, /Play next/)
  assert.doesNotMatch(html, /audio-dismiss/)
  assert.doesNotMatch(html, /--progress:/)
})

test('spoken media keeps meaningful resume progress', () => {
  const html = view().row({ library_id:'books', id:'1', library_kind:'audiobooks', title:'Chapter', progress:{ position_seconds:50, duration_seconds:100 } })
  assert.match(html, /--progress:50%/)
})

test('compact discovery rows can omit playback progress without losing metadata', () => {
  const html = view().row({ library_id:'music', id:'1', library_kind:'music', title:'Track', artist:'Artist', progress:{ position_seconds:50, duration_seconds:100 } }, false, false)
  assert.match(html, /Track/)
  assert.match(html, /Artist/)
  assert.doesNotMatch(html, /--progress:/)
})

test('music cover cards retain artwork and the complete track action menu', () => {
  const html = view({ isFavorite: () => true }).musicCard({ library_id:'music', id:'1', library_kind:'music', title:'A & B', artist:'Artist', album:'Album', artwork_url:'/art?x=1&y=2' })
  assert.match(html, /music-track-card/)
  assert.match(html, /data-art="\/art\?x=1&amp;y=2"/)
  assert.match(html, /Remove favorite/)
  assert.match(html, /Add to queue/)
  assert.match(html, /<summary aria-label="More options for A &amp; B">/)
})

test('album rows use disc and track numbers without repeated artwork', () => {
  const html = view().musicAlbumRow({ library_id:'music', id:'1', library_kind:'music', title:'Second disc song', artist:'Artist', disc:2, track:3, duration_seconds:125 }, 8)
  assert.match(html, /music-track-number[^>]*>2·3</)
  assert.match(html, /Second disc song/)
  assert.match(html, /2:05/)
  assert.match(html, /data-action="next"/)
  assert.doesNotMatch(html, /audio-glyph|data-art=/)
})

test('playlist tracks reuse their options menu for ordering and removal', () => {
  const html = view({ browse: () => 'playlists' }).row({ library_id:'music', id:'1', library_kind:'music', title:'Track' })
  assert.match(html, /data-action="playlist-up"/)
  assert.match(html, /data-action="playlist-down"/)
  assert.match(html, /data-action="playlist-remove"/)
})

test('queue tracks retain download and information actions', () => {
  const html = view({ browse: () => 'queue' }).row({ library_id:'music', id:'1', library_kind:'music', title:'Track', download_url:'/audio/music/1/Track.flac?download=1' })
  assert.match(html, /data-action="queue-remove"/)
  assert.match(html, /href="\/audio\/music\/1\/Track\.flac\?download=1" download/)
  assert.match(html, /data-action="track-info"/)
})

test('dashboard rows expose only the explicit continue-listening dismissal', () => {
  const html = view().row({ library_id:'music', id:'1', library_kind:'music', title:'Track' }, true)
  assert.match(html, /audio-dismiss/)
  assert.doesNotMatch(html, /audio-more/)
})

test('podcast rows retain descriptions, publication details, and download action', () => {
  const html = view().row({ library_id:'podcasts', id:'1', library_kind:'podcasts', title:'Episode', description:'A <story>', modified:1, size:1048576, download_url:'/episode' })
  assert.match(html, /A &lt;story&gt;/)
  assert.match(html, /2 days ago · 1\.0 MB/)
  assert.match(html, /audio-quick-download/)
})

test('a podcast episode says plainly whether the server has it, and the arrow saves it there', () => {
  const episode = { library_id:'podcasts', id:'9', library_kind:'podcasts', title:'Episode 9', modified:1, size:1048576, download_url:'/episode' }
  const missing = view().row(episode)
  assert.doesNotMatch(missing, /On server/)
  assert.match(missing, /data-action="server-save" data-state="idle"/)
  assert.match(missing, /class="audio-quick-download audio-quick-device" data-state="idle" href="\/episode" download/, 'saving to the device stays available')
  const saved = view().row({ ...episode, server_downloaded: true })
  assert.match(saved, /2 days ago · 1\.0 MB<\/small>/)
  assert.match(saved, /<em class="episode-badge" data-kind="server">On server<\/em>/)
  assert.match(saved, /data-state="ready"/)
})

test('a song that has ended is not marked Played or greyed out, but a finished podcast episode and audiobook still are', () => {
  const finished = { position_seconds: 100, duration_seconds: 100, finished: true }
  const song = view().row({ library_id: 'music', id: '1', library_kind: 'music', title: 'Song', artist: 'Artist', duration_seconds: 125, progress: finished })
  assert.doesNotMatch(song, /Played/)
  assert.doesNotMatch(song, /is-played/)
  for (const kind of ['podcasts', 'audiobooks']) {
    const html = view().row({ library_id: kind, id: '1', library_kind: kind, title: 'Episode', duration_seconds: 125, progress: finished })
    assert.match(html, /is-played/, kind)
  }
})
