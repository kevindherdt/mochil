import assert from 'node:assert/strict'
import test from 'node:test'
import { railCountText, setRailCount } from '../static/rail-count.js'

test('a rail says how many items it holds, in the design\'s words', () => {
  assert.equal(railCountText(4), '4 items')
  assert.equal(railCountText(1), '1 item')
  assert.equal(railCountText(0), '')
  assert.equal(railCountText(undefined), '')
})

test('an empty rail shows nothing rather than "0 items"', () => {
  const node = { textContent: 'stale' }
  const doc = { getElementById: () => node }
  setRailCount(doc, 'jump-count', 0)
  assert.equal(node.textContent, '')
})

test('a missing header is not an error', () => {
  setRailCount({ getElementById: () => null }, 'nope', 3)
  setRailCount(undefined, 'nope', 3)
})
