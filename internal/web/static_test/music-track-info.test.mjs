import assert from 'node:assert/strict'
import test from 'node:test'

import { musicTrackDetails, musicTrackInfoHTML } from '../static/music-track-info.js'

const escapeHTML = value => String(value).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('"','&quot;')

test('track details show known metadata and omit unknown values',()=>{
  assert.deepEqual(musicTrackDetails({
    artist:'Paramore',album_artist:'Paramore',album:'Riot!',genre:'Rock',year:2007,
    disc:1,track:4,container:'flac',bitrate:987654,duration_seconds:219.6,size:31_457_280,
  }),[
    ['Artist','Paramore'],['Album','Riot!'],['Genre','Rock'],['Year','2007'],
    ['Disc','1'],['Track','4'],['Format','FLAC'],['Bitrate','988 kbps'],
    ['Duration','3:40'],['File size','30.0 MB'],
  ])
  assert.deepEqual(musicTrackDetails({artist:'Unknown artist'}),[])
})

test('track information markup escapes catalogue metadata',()=>{
  const html=musicTrackInfoHTML({title:'<Track>',artist:'A & B',container:'mp3'},escapeHTML)
  assert.match(html,/id="music-track-info-title"/)
  assert.match(html,/&lt;Track>/)
  assert.match(html,/A &amp; B/)
  assert.match(html,/MP3/)
  assert.doesNotMatch(html,/<Track>/)
})
