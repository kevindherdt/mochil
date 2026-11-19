import assert from 'node:assert/strict'
import test from 'node:test'
import { createDestinationRail, destinationAfterKey, destinationAfterSwipe, railOverflow, railScrollTarget } from '../static/destination-rail.js'

const destinations = [['series', 'Series'], ['authors', 'Authors'], ['books', 'All']]

test('shared destination swipe stays bounded', () => {
  assert.equal(destinationAfterSwipe(destinations, 'series', -90), 'authors')
  assert.equal(destinationAfterSwipe(destinations, 'books', -90), 'books')
  assert.equal(destinationAfterSwipe(destinations, 'authors', 90), 'series')
})

test('shared rail helpers preserve viewport context', () => {
  assert.deepEqual(railOverflow(20, 100, 160), { before: true, after: true })
  assert.equal(railScrollTarget({ scrollLeft: 20, clientWidth: 100 }, { offsetLeft: 8, offsetWidth: 40 }), 0)
})

test('keyboard navigation activates a bounded neighboring destination', () => {
  assert.equal(destinationAfterKey(destinations, 'series', 'ArrowRight'), 'authors')
  assert.equal(destinationAfterKey(destinations, 'authors', 'ArrowLeft'), 'series')
  assert.equal(destinationAfterKey(destinations, 'series', 'End'), 'books')
  assert.equal(destinationAfterKey(destinations, 'books', 'Home'), 'series')
  assert.equal(destinationAfterKey(destinations, 'books', 'ArrowRight'), 'books')
})

test('every destination implementation receives the shared visual primitive', () => {
  const classes = new Set()
  const element = {
    classList: { add: value => classes.add(value), toggle() {}, remove() {} },
    innerHTML: '',
    addEventListener() {},
    querySelectorAll() { return [] },
  }
  createDestinationRail({
    element,
    destinations,
    attribute: 'data-test-destination',
    getValue: () => 'series',
    setValue() {},
    async onChange() {},
  })
  assert.equal(classes.has('destination-rail'), true)
})
