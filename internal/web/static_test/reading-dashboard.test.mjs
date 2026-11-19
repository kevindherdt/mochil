import assert from 'node:assert/strict'
import test from 'node:test'

import { dashboardReadingTitle } from '../static/reading-dashboard.js'

test('chapter-only dashboard titles include their series', () => {
  assert.equal(
    dashboardReadingTitle({ title: 'Chapter 0', series: 'Battle Royale' }),
    'Battle Royale — Chapter 0',
  )
})

test('descriptive publication titles remain unchanged', () => {
  assert.equal(
    dashboardReadingTitle({ title: 'The Great Hunt', series: 'The Wheel of Time' }),
    'The Great Hunt',
  )
})

test('dashboard titles handle missing catalogue text', () => {
  assert.equal(dashboardReadingTitle({ title: '', series: 'Example' }), 'Untitled')
})
