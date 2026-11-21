import assert from 'node:assert/strict'
import test from 'node:test'
import { readFile } from 'node:fs/promises'

// The rule lives inside the module's factory, so it is lifted out here and run for real rather than grepped.
const source = await readFile(new URL('../static/book-cards.js', import.meta.url), 'utf8')
const body = source.slice(source.indexOf('function displayTitle'), source.indexOf('function bookMarkup'))
const { displayTitle, seriesIndex, seriesIndexLabel } = new Function(
  `${body}; return { displayTitle, seriesIndex, seriesIndexLabel }`,
)()

test('a zero-padded series index is not part of the title', () => {
  assert.equal(displayTitle({ title: '01 Dies the Fire' }), 'Dies the Fire')
  assert.equal(displayTitle({ title: '02 Academ\'s Fury' }), "Academ's Fury")
  assert.equal(displayTitle({ title: '03 - Abaddon\'s Gate' }), "Abaddon's Gate")
})

test('a number that is the title stays the title', () => {
  assert.equal(displayTitle({ title: '1984' }), '1984')
  assert.equal(displayTitle({ title: '127 Hours' }), '127 Hours')
  assert.equal(displayTitle({ title: 'The 7 Habits' }), 'The 7 Habits')
})

test('an ordinary title is untouched, and an empty one is named', () => {
  assert.equal(displayTitle({ title: 'Neuromancer' }), 'Neuromancer')
  assert.equal(displayTitle({}), 'Untitled')
})

test('the index is kept, and shown on its own label', () => {
  assert.equal(seriesIndex({ title: '01 Dies the Fire' }), 1)
  assert.equal(seriesIndexLabel({ title: '01 Dies the Fire' }), 'Book 1')
  assert.equal(seriesIndexLabel({ title: 'The Age of Ra' }), '')
})

test('a stored series index wins over one guessed from the title', () => {
  assert.equal(seriesIndex({ title: 'Dies the Fire', series: 'The Change', series_index: 3 }), 3)
  assert.equal(seriesIndexLabel({ title: 'Dies the Fire', series: 'The Change', series_index: 3 }), 'Book 3')
})

test('a stored index without a series is noise, not a position', () => {
  assert.equal(seriesIndex({ title: '1984', series_index: 1 }), 0)
  assert.equal(seriesIndexLabel({ title: '1984', series_index: 1 }), '')
})

test('a number that is the title is not an index', () => {
  assert.equal(seriesIndex({ title: '1984' }), 0)
  assert.equal(seriesIndexLabel({ title: '1984' }), '')
})

test('a series named after the book is not a series', () => {
  assert.equal(seriesIndex({ title: '1984', series: '1984', series_index: 1 }), 0)
  assert.equal(seriesIndexLabel({ title: '1984', series: '1984', series_index: 1 }), '')
})
