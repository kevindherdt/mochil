import assert from 'node:assert/strict'
import test from 'node:test'
import { destinationAfterSwipe } from '../static/destination-rail.js'

const destinations = [['series', 'Series'], ['books', 'All'], ['collections', 'Collections']]

test('publication destinations use the shared bounded swipe order', () => {
  assert.equal(destinationAfterSwipe(destinations, 'series', -90), 'books')
  assert.equal(destinationAfterSwipe(destinations, 'books', -90), 'collections')
  assert.equal(destinationAfterSwipe(destinations, 'collections', -90), 'collections')
})
