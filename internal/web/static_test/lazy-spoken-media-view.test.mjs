import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('cold Music keeps podcast and audiobook templates behind a lazy boundary', async () => {
  const library = await readFile(new URL('../static/audio-library.js', import.meta.url), 'utf8')
  const facade = await readFile(new URL('../static/lazy-spoken-media-view.js', import.meta.url), 'utf8')
  assert.doesNotMatch(library, /^import .*audio-media-views/m)
  assert.match(library, /^import .*lazy-spoken-media-view/m)
  assert.match(facade, /import\('\.\/audio-media-views\.js\?v=5'\)/)
  assert.match(library, /if\(kind==='podcasts'\|\|kind==='audiobooks'\)/)
  assert.match(library, /if\(generation!==routeID\)return false/)
})
