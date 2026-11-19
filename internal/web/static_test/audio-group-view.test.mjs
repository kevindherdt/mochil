import assert from 'node:assert/strict'
import test from 'node:test'
import { audioGroupLabel, groupedAudioHTML } from '../static/audio-group-view.js'

test('audio groups use media-specific labels', () => {
  assert.equal(audioGroupLabel({ album: 'Episode show' }, 'podcasts'), 'Episode show')
  assert.equal(audioGroupLabel({ series: 'Saga' }, 'audiobooks'), 'Saga')
  assert.equal(audioGroupLabel({ album_artist: 'Artist' }, 'music', 'artists'), 'Artist')
})

test('grouped audio markup preserves grouping and loaded context', () => {
  const items = [{ title: 'One', album: 'A' }, { title: 'Two', album: 'A' }]
  const html = groupedAudioHTML(items, {
    label: item => item.album,
    row: item => `<article>${item.title}</article>`,
    escapeHTML: String,
    music: true,
    total: 4,
  })
  assert.match(html, /<h2>A<\/h2><span>2 loaded<\/span>/)
  assert.match(html, /data-play-group="A"/)
  assert.equal((html.match(/<article>/g) || []).length, 2)
})
