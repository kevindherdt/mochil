import assert from 'node:assert/strict'
import test from 'node:test'
import { visibleVideoContinuations } from '../static/video-continue-rail.js'

test('continue watching remains visible, playable, unfinished, and bounded', () => {
  const progress = { position_seconds:30, duration_seconds:100 }
  const items = [
    { id:'one', library_id:'video', stream_url:'/one', progress },
    { id:'hidden', library_id:'hidden', stream_url:'/hidden', progress },
    { id:'finished', library_id:'video', stream_url:'/finished', progress:{ ...progress, finished:true } },
    { id:'dismissed', library_id:'video', hls_url:'/dismissed', progress:{ ...progress, dismissed:true } },
    { id:'unplayable', library_id:'video', progress },
    { id:'two', library_id:'video', hls_url:'/two', progress },
  ]
  assert.deepEqual(visibleVideoContinuations(items, library => library === 'video', 1).map(item => item.id), ['one'])
  assert.deepEqual(visibleVideoContinuations(items, library => library === 'video').map(item => item.id), ['one', 'two'])
})

test('invalid continuation payloads stay empty', () => {
  assert.deepEqual(visibleVideoContinuations(null, () => true), [])
})
