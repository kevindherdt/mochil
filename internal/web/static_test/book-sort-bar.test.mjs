import test from 'node:test'
import assert from 'node:assert/strict'
import { readBookSort } from '../static/book-sort-bar.js'

test('the book sort choice is only one the server understands', () => {
  assert.equal(readBookSort({ getItem: () => 'recent' }), 'recent')
  assert.equal(readBookSort({ getItem: () => 'author' }), 'author')
  assert.equal(readBookSort({ getItem: () => 'rating' }), '')
  assert.equal(readBookSort({ getItem: () => { throw new Error('blocked') } }), '')
})
