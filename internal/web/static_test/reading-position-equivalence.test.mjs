import assert from 'node:assert/strict'
import test from 'node:test'
import { readingPositionsEquivalent } from '../static/reading-position-equivalence.js'

test('reflowable semantic anchors survive renderer fraction differences', () => {
  assert.equal(readingPositionsEquivalent({ fraction: .2, cfi: 'same' }, { fraction: .8, cfi: 'same' }), true)
  assert.equal(readingPositionsEquivalent({ fraction: .2, koreader_locator: '/same' }, { fraction: .8, koreader_locator: '/same' }), true)
  const web_anchor = { spine_index: 4, prefix: 'before', exact: 'same sentence', suffix: 'after' }
  assert.equal(readingPositionsEquivalent({ fraction: .2, web_anchor }, { fraction: .8, web_anchor: { ...web_anchor, progress: .9 } }), true)
})

test('fixed layout remains exact and ambiguous anchors are rejected', () => {
  assert.equal(readingPositionsEquivalent({ fraction: .2, cfi: 'same', locators: { comic: '{"page":4}' } }, { fraction: .2, cfi: 'same', locators: { pdf: '{"page":5}' } }), false)
  assert.equal(readingPositionsEquivalent({ fraction: .2, web_anchor: { spine_index: 1, exact: 'repeat' } }, { fraction: .8, web_anchor: { spine_index: 1, exact: 'repeat' } }), false)
  assert.equal(readingPositionsEquivalent({ fraction: .2 }, { fraction: .21 }), false)
})

