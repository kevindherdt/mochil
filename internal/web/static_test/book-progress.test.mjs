import assert from 'node:assert/strict'
import test from 'node:test'
import { bookReadingState } from '../static/book-progress.js'

test('publication progress exposes a concise percentage', () => {
  assert.deepEqual(bookReadingState({ reading_progress: { fraction: 0.478 } }), {
    label: '48% read', percentage: 48, finished: false,
  })
})

test('finished publications are visibly complete even with an earlier locator', () => {
  assert.deepEqual(bookReadingState({ reading_progress: { fraction: 0.82, finished: true } }), {
    label: 'Read', percentage: 100, finished: true,
  })
})

test('untouched publications do not gain noisy progress chrome', () => {
  assert.equal(bookReadingState({}).label, '')
})

test('every status has its own word, unread included', () => {
  // Only the two parked states were mapped, so anything else fell through to 'Dropped'.
  assert.equal(bookReadingState({ reading_progress: { status: 'unread', fraction: 0.42 } }).label, 'Unread')
  assert.equal(bookReadingState({ reading_progress: { status: 'on_hold' } }).label, 'On hold')
  assert.equal(bookReadingState({ reading_progress: { status: 'dropped' } }).label, 'Dropped')
})
