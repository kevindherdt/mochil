import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { musicGroupArtworkURL, musicGroupGridHTML, musicGroupMonogram, musicTrackGroupHeadingHTML } from '../static/music-group-view.js'

const escapeHTML = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;')

test('music group artwork encodes server-controlled identifiers', () => {
  assert.equal(musicGroupArtworkURL({ artwork_library_id:'music one', artwork_item_id:'a/b' }), '/api/audio-artwork/music%20one/a%2Fb')
  assert.equal(musicGroupArtworkURL({}), '')
})

test('music group grid renders compact escaped collection cards', () => {
  const html = musicGroupGridHTML([{ name:'Album <One>', subtitle:'Artist & Co', count:2, artwork_library_id:'music', artwork_item_id:'cover' }], { escapeHTML, type:'albums' })
  assert.match(html, /data-music-group="Album &lt;One&gt;"/)
  assert.match(html, /Artist &amp; Co/)
  assert.match(html, /data-art="\/api\/audio-artwork\/music\/cover"/)
  assert.match(html, /aria-hidden="true">AO</)
})

test('music group fallbacks remain recognizable without artwork', () => {
  assert.equal(musicGroupMonogram('All We Know Is Falling'), 'AF')
  assert.equal(musicGroupMonogram('IU'), 'IU')
  assert.equal(musicGroupMonogram(''), '♪')
  const html = musicGroupGridHTML([{ name:'No Cover', count:1 }], { escapeHTML, type:'albums' })
  assert.match(html, /music-group-art/)
  assert.match(html, />NC<\/b>/)
  assert.doesNotMatch(html, /data-art=/)
})

test('nested artist and album headings expose one clear back action', () => {
  const grid = musicGroupGridHTML([], { escapeHTML, type:'albums', title:'Artist', backLabel:'Back to artists', layout:'list' })
  const tracks = musicTrackGroupHeadingHTML({ id:'opaque/id', name:'Album', count:3, artwork_library_id:'music one', artwork_item_id:'a/b' }, 'Artist', escapeHTML)
  assert.equal((grid.match(/data-music-group-back/g) || []).length, 1)
  assert.equal((grid.match(/<svg /g) || []).length, 1)
  assert.equal((tracks.match(/<svg /g) || []).length, 1)
  assert.doesNotMatch(`${grid}${tracks}`, /←/)
  assert.match(grid, /music-group-grid list/)
  assert.match(tracks, /Artist · 3 tracks/)
  assert.match(tracks, /music-group-heading-art/)
  assert.match(tracks, /\/api\/audio-artwork\/music%20one\/a%2Fb/)
  assert.match(tracks, /data-play-music-group/)
  assert.match(tracks, /data-shuffle-music-group aria-label="Shuffle Album"/)
  assert.match(tracks, /audio-group-download\?library=music%20one&amp;group_id=opaque%2Fid/)
})

test('focused album titles remain complete while actions reflow at compact widths', () => {
  const css = readFileSync(new URL('../static/music-library.css', import.meta.url), 'utf8')
  assert.match(css, /\.music-group-heading\.featured h2\{[^}]*overflow:visible;[^}]*text-overflow:clip;[^}]*white-space:normal/)
  assert.doesNotMatch(css, /\.music-group-heading\.featured h2\{[^}]*line-clamp/)
  assert.match(css, /@media\(max-width:900px\)\{\.music-group-heading\.featured\{grid-template-columns:44px 88px minmax\(0,1fr\)\}\.music-group-actions\{grid-column:3\/4/)
  assert.match(css, /@media\(max-width:560px\)\{\.music-group-heading\.featured\{[^}]*grid-template-columns:44px 64px minmax\(0,1fr\)/)
})
