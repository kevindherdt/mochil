import assert from 'node:assert/strict'
import test from 'node:test'
import { createMediaSeekOrigin } from '../static/media-seek-origin.js'

test('matching player restoration is passive and consumed once', () => {
  const origin = createMediaSeekOrigin()
  origin.expectAutomatic(42)
  assert.equal(origin.isExplicit(42.4), false)
  assert.equal(origin.isExplicit(42.4), true)
})

test('a different or cleared browser seek remains explicit', () => {
  const origin = createMediaSeekOrigin()
  origin.expectAutomatic(42)
  assert.equal(origin.isExplicit(12), true)
  origin.expectAutomatic(42)
  origin.clear()
  assert.equal(origin.isExplicit(42), true)
})
