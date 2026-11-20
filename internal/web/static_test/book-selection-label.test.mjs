import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { selectionLabel } from '../static/book-selection.js'

// The batch bar used to say 'Select all' while only marking the cards that were
// loaded (120 of 5,430 on the live shelf). These pin the honest label.
test('the select control names the loaded count it would take', () => {
  assert.equal(selectionLabel(120).text, 'Select loaded (120)')
  assert.equal(selectionLabel(5430).text, 'Select loaded (5430)')
})

test('the label says books beyond the window are not selected', () => {
  const title = selectionLabel(120).title
  assert.match(title, /120 books loaded on screen/)
  assert.match(title, /not selected/)
})

test('singular and empty shelves read correctly', () => {
  assert.equal(selectionLabel(1).text, 'Select loaded (1)')
  assert.match(selectionLabel(1).title, /the 1 book loaded/)
  assert.equal(selectionLabel(0).text, 'Select loaded (0)')
  assert.match(selectionLabel(0).title, /0 books loaded/)
})

test('a nonsense count cannot invent a selection size', () => {
  assert.equal(selectionLabel(undefined).text, 'Select loaded (0)')
  assert.equal(selectionLabel('many').text, 'Select loaded (0)')
  assert.equal(selectionLabel(-3).text, 'Select loaded (0)')
})

test('the bar no longer offers a bare Select all', () => {
  const source = readFileSync(new URL('../static/book-selection.js', import.meta.url), 'utf8')
  assert.doesNotMatch(source, /data-act="all">Select all</)
  assert.match(source, /data-act="all">Select loaded \(0\)</)
})
