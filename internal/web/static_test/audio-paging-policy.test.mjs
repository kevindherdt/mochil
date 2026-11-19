import assert from 'node:assert/strict'
import test from 'node:test'
import { audioPagingDecision, canLoadPreviousMusicPage } from '../static/audio-paging-policy.js'

test('music paging distinguishes finite views, tracks, and grouped catalogues', () => {
  assert.deepEqual(
    audioPagingDecision({ visible: true, kind: 'music', browse: 'favorites', itemCount: 2, total: 20 }),
    { canLoad: false, target: 'audio-items', autoLoad: false },
  )
  assert.deepEqual(
    audioPagingDecision({ visible: true, kind: 'music', browse: 'favorites', music: { localHasMore: true } }),
    { canLoad: true, target: 'local-music', autoLoad: true },
  )
  assert.deepEqual(
    audioPagingDecision({ visible: true, kind: 'music', browse: 'tracks', itemCount: 72, startOffset: 72, total: 300 }),
    { canLoad: true, target: 'audio-items', autoLoad: true },
  )
  assert.deepEqual(
    audioPagingDecision({ visible: true, kind: 'music', browse: 'albums', music: { count: 72, total: 100, nextOffset: 72 } }),
    { canLoad: true, target: 'music-groups', autoLoad: true },
  )
})

test('spoken-media paging requires an open group and idle catalogue', () => {
  assert.equal(audioPagingDecision({ visible: true, kind: 'podcasts', podcast: { show: 'Show', count: 20, total: 30, loading: false } }).canLoad, true)
  assert.equal(audioPagingDecision({ visible: true, kind: 'podcasts', podcast: { show: '', count: 0, total: 30, loading: false } }).canLoad, false)
  assert.deepEqual(
    audioPagingDecision({ visible: true, kind: 'audiobooks', browse: 'series', audiobook: { group: 'Series', count: 10, total: 11, loading: false } }),
    { canLoad: true, target: 'audiobook-group', autoLoad: false },
  )
  assert.equal(audioPagingDecision({ visible: true, kind: 'audiobooks', browse: 'recent', itemCount: 10, total: 30 }).autoLoad, true)
})

test('hidden surfaces and busy catalogues never request another page', () => {
  assert.equal(audioPagingDecision({ visible: false, kind: 'music', browse: 'tracks', total: 30 }).canLoad, false)
  assert.equal(audioPagingDecision({ visible: true, kind: 'music', browse: 'tracks', total: 30, loading: true }).canLoad, false)
  assert.equal(canLoadPreviousMusicPage({ visible: true, kind: 'music', browse: 'tracks', startOffset: 72 }), true)
  assert.equal(canLoadPreviousMusicPage({ visible: true, kind: 'music', browse: 'albums', startOffset: 72 }), false)
})
