import assert from 'node:assert/strict'
import test from 'node:test'
import { lookupPool } from '../static/audio-item-lookup.js'

const discovery = { played: ['p'], recent: ['r'], favorites: ['f'] }
const base = { discovery, playlistItems: ['list'], visibleItems: () => ['episode'] }

test('music discovery and playlists look rows up in their own lists', () => {
  assert.deepEqual(lookupPool({ ...base, kind: 'music', browse: 'discover' }), ['p', 'r', 'f'])
  assert.deepEqual(lookupPool({ ...base, kind: 'music', browse: 'playlists' }), ['list'])
  assert.deepEqual(lookupPool({ ...base, kind: 'music', browse: 'tracks' }), ['episode'])
})

test('a lingering music browse value does not hijack podcasts or audiobooks', () => {
  for (const kind of ['podcasts', 'audiobooks']) {
    for (const browse of ['discover', 'playlists']) {
      assert.deepEqual(lookupPool({ ...base, kind, browse }), ['episode'], `${kind} with music browse "${browse}"`)
    }
  }
})
