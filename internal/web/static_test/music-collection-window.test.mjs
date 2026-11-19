import assert from 'node:assert/strict'
import test from 'node:test'
import { createMusicCollectionWindow } from '../static/music-collection-window.js'

test('finite music collections grow in bounded automatic windows', () => {
  let renders = 0
  const window = createMusicCollectionWindow({ pageSize: 120, onMore: () => { renders += 1 } })
  assert.equal(window.limit, 120)
  window.apply({ hasMore: true })
  assert.equal(window.more(), true)
  assert.equal(window.limit, 240)
  assert.equal(renders, 1)
  window.apply({ hasMore: false })
  assert.equal(window.more(), false)
  window.reset()
  assert.equal(window.limit, 120)
  assert.equal(window.hasMore, false)
})
