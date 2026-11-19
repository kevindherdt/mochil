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
