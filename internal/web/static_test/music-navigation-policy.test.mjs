import assert from 'node:assert/strict'
import test from 'node:test'
import { musicAlphabetEnabled, musicEffectiveSort, musicNavigationEntries, musicNavigationLabel, musicTitleBucket } from '../static/music-navigation-policy.js'

test('music alphabet appears only on name-ordered catalogue destinations', () => {
  for (const browse of ['albums', 'artists']) {
    assert.equal(musicAlphabetEnabled({ browse }), true, browse)
  }
  assert.equal(musicAlphabetEnabled({ browse: 'tracks', sort: 'title' }), true)
  assert.equal(musicAlphabetEnabled({ browse: 'tracks', sort: 'artist' }), true)
  assert.equal(musicAlphabetEnabled({ browse: 'tracks', sort: 'recent' }), false)
  assert.equal(musicAlphabetEnabled({ browse: 'tracks', sort: 'oldest' }), false)
  for (const browse of ['discover', 'recently-played', 'favorites', 'playlists', 'queue']) {
    assert.equal(musicAlphabetEnabled({ browse }), false, browse)
  }
})

test('track navigation follows the field that actually owns the active ordering', () => {
  assert.equal(musicNavigationLabel({ browse: 'tracks', sort: 'title', title: 'Crush', context: 'Paramore · Riot!' }), 'Crush')
  assert.equal(musicNavigationLabel({ browse: 'tracks', sort: 'artist', title: 'Crush', context: 'Paramore · Riot!' }), 'Paramore')
  assert.equal(musicNavigationLabel({ browse: 'tracks', sort: 'artist', title: 'Instrumental', context: '' }), 'Instrumental')
})

test('navigation entries retain their elements while deriving artist labels from row context', () => {
  const element = {
    querySelector(selector) {
      return { textContent: selector === 'h2,strong' ? 'Crush' : 'Paramore · Riot! · 3:10' }
    },
  }
  assert.deepEqual(musicNavigationEntries([element], { browse: 'tracks', sort: 'artist' }), [{ element, label: 'Paramore' }])
})

test('music detail surfaces do not retain a misleading catalogue alphabet', () => {
  assert.equal(musicAlphabetEnabled({ browse: 'albums', selectedGroup: { name: 'Album' } }), false)
  assert.equal(musicAlphabetEnabled({ browse: 'artists', selectedArtist: { name: 'Artist' } }), false)
})

test('music navigation owns its server sort and article-insensitive buckets', () => {
  assert.equal(musicEffectiveSort('music', 'albums', 'title'), 'album')
  assert.equal(musicEffectiveSort('music', 'artists', 'recent'), 'artist')
  assert.equal(musicEffectiveSort('audiobooks', 'artists', 'recent'), 'recent')
  assert.equal(musicTitleBucket('The Beatles'), 'B')
  assert.equal(musicTitleBucket('86'), '#')
})
