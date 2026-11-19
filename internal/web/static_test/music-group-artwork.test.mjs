import assert from 'node:assert/strict'
import test from 'node:test'
import { musicGroupArtworkURL, musicGroupPlaybackTracks } from '../static/music-group-artwork.js'

test('group artwork safely encodes server-controlled identifiers', () => {
  assert.equal(musicGroupArtworkURL({ artwork_library_id:'music one', artwork_item_id:'a/b' }), '/api/audio-artwork/music%20one/a%2Fb')
  assert.equal(musicGroupArtworkURL(), '')
})

test('focused album tracks inherit player-only artwork', () => {
  const tracks = musicGroupPlaybackTracks([{ id:'one' }], { artwork_library_id:'music', artwork_item_id:'cover' })
  assert.equal(tracks[0].player_artwork_url, '/api/audio-artwork/music/cover')
  assert.equal(tracks[0].artwork_url, undefined)
})

test('track artwork remains authoritative and malformed results stay bounded', () => {
  const track = { id:'one', artwork_url:'/track-cover' }
  assert.equal(musicGroupPlaybackTracks([track], { artwork_library_id:'music', artwork_item_id:'album' })[0], track)
  assert.deepEqual(musicGroupPlaybackTracks(null, {}), [])
})
