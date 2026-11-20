import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('track information dialog and styles stay outside cold Music', async () => {
  const root = new URL('../static/', import.meta.url)
  const [audio, deferred, lazy] = await Promise.all([
    readFile(new URL('audio-library.js', root), 'utf8'),
    readFile(new URL('deferred-audio-controller.js', root), 'utf8'),
    readFile(new URL('lazy-music-track-info.js', root), 'utf8'),
  ])
  assert.doesNotMatch(audio, /from '.\/music-track-info\.js/)
  assert.doesNotMatch(deferred, /music-track-info\.css/)
  assert.match(audio, /createLazyMusicTrackInfo/)
  assert.match(lazy, /import\('\.\/music-track-info\.js/)
  assert.match(lazy, /music-track-info\.css/)
})
