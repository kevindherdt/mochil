import test from 'node:test'
import assert from 'node:assert/strict'
import { createBucketTracker, visibleCatalogueBucket } from '../static/catalog-scroll-position.js'

const item = (bucket, bottom) => ({ dataset: { bucket }, getBoundingClientRect: () => ({ bottom }) })

test('scroll tracking reports the first catalogue item below the sticky content edge', () => {
  assert.equal(visibleCatalogueBucket([item('A', 80), item('B', 130), item('C', 200)], 100), 'B')
})

test('scroll tracking has safe fallbacks at the end and for symbol buckets', () => {
  assert.equal(visibleCatalogueBucket([item('', 120)], 100), '#')
  assert.equal(visibleCatalogueBucket([item('Z', 80)], 100), '')
})

const card = (bucket, bottom) => ({ dataset: { bucket }, getBoundingClientRect: () => ({ bottom }) })

test('a tracker walks from its last match instead of reading every card', () => {
  const reads = { count: 0 }
  const items = Array.from({ length: 600 }, (_, index) => ({
    dataset: { bucket: 'ABCDEFGHIJ'[Math.floor(index / 60)] || 'Z' },
    getBoundingClientRect() {
      reads.count += 1
      return { bottom: 100 + index * 444 }
    },
  }))
  const tracker = createBucketTracker()
  assert.equal(tracker.find(items, 106), 'A')
  reads.count = 0
  // Scrolling a page (240 px, less than one 444 px row) re-reads a row or two.
  for (let step = 1; step <= 100; step += 1) tracker.find(items, 106 + step * 240)
  assert.ok(reads.count <= 350, `read ${reads.count} rects for 100 scroll frames`)
  // The bucket reported after that scroll is the true one, not the remembered one.
  const top = 106 + 100 * 240
  assert.equal(tracker.find(items, top), visibleCatalogueBucket(items, top))
})

test('a tracker agrees with the stateless lookup at every position', () => {
  const items = Array.from({ length: 40 }, (_, index) => card('ABCDEFGHIJ'[index % 10], 100 + index * 60))
  const tracker = createBucketTracker()
  for (const top of [0, 90, 130, 400, 1000, 2500, 4000]) {
    assert.equal(tracker.find(items, top), visibleCatalogueBucket(items, top), `top ${top}`)
  }
})

test('a tracker keeps the safe fallbacks and survives a rebuilt list', () => {
  const tracker = createBucketTracker()
  const list = [card('A', 80), card('B', 130), card('C', 200)]
  assert.equal(tracker.find(list, 100), 'B')
  assert.equal(tracker.find([], 100), '')
  assert.equal(tracker.find([card('', 120)], 100), '#')
  // A reloaded shelf hands over a shorter list; the remembered index clamps.
  assert.equal(tracker.find([card('D', 150)], 100), 'D')
  assert.equal(tracker.find([card('D', 80)], 100), '')
})
