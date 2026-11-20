import test from 'node:test'
import assert from 'node:assert/strict'
import { applyOrder, moveBefore } from '../static/editor-column-order.js'
import { badgeHTML, canFetch, toneOf } from '../static/editor-badges.js'

test('columns follow the saved order and unknown ones go last', () => {
  const cols = [{ id: 'a' }, { id: 'b' }, { id: 'c' }]
  assert.deepEqual(applyOrder(cols, ['c', 'a']).map(c => c.id), ['c', 'a', 'b'])
  assert.deepEqual(applyOrder(cols, []).map(c => c.id), ['a', 'b', 'c'])
})

test('moving a column puts it before the target or last', () => {
  assert.deepEqual(moveBefore(['a', 'b', 'c'], 'c', 'a'), ['c', 'a', 'b'])
  assert.deepEqual(moveBefore(['a', 'b', 'c'], 'a', ''), ['b', 'c', 'a'])
})

test('missing values are red and can be fetched; duplicates are amber', () => {
  assert.equal(toneOf('no_isbn'), 'bad')
  assert.equal(toneOf('duplicate'), 'warn')
  assert.equal(toneOf('side_story'), 'quiet')
  assert.ok(canFetch('no_isbn') && !canFetch('duplicate'))
  assert.match(badgeHTML('no_isbn', 'No ISBN', 'r1', String), /data-tone="bad".*data-fetch="r1"/)
  assert.doesNotMatch(badgeHTML('duplicate', 'Duplicate number', 'r1', String), /data-fetch/)
})
