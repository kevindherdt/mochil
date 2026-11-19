import test from 'node:test'
import assert from 'node:assert/strict'
import { musicArtworkMonogram } from '../static/music-artwork-fallback.js'

test('music artwork fallbacks remain compact and language-safe', () => {
  assert.equal(musicArtworkMonogram('All We Know Is Falling'), 'AF')
  assert.equal(musicArtworkMonogram('Paramore'), 'PA')
  assert.equal(musicArtworkMonogram('菊池桃子'), '菊池')
  assert.equal(musicArtworkMonogram(''), '♪')
})
