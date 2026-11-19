import assert from 'node:assert/strict'
import test from 'node:test'
import { blockedText, confirmText, doneText, transferRequest, transferVerb } from '../static/editor-transfer.js'

test('the request says what to do and whether to only look', () => {
  assert.deepEqual(transferRequest('books', 'shelf', ['a'], 'move', true), { library: 'books', to: 'shelf', ids: ['a'], mode: 'move', preview: true })
  assert.equal(transferVerb('copy').done, 'Copied')
})

test('problems are named briefly and say that nothing was changed', () => {
  const preview = { books: [{ title: 'A', reason: 'Taken.' }, { title: 'B' }, { title: 'C', reason: 'Gone.' }, { title: 'D', reason: 'Gone.' }, { title: 'E', reason: 'Gone.' }] }
  const text = blockedText(preview, 2)
  assert.match(text, /4 books cannot go/)
  assert.match(text, /“A”: Taken\./)
  assert.match(text, /and 2 more/)
  assert.match(text, /Nothing was changed/)
  assert.equal(blockedText({ books: [{ title: 'B' }] }), '')
})

test('a move explains what travels with the book, a copy what it does not', () => {
  assert.match(confirmText('move', 1, 'Shelf'), /Move 1 book to “Shelf”.*reading place go with them/)
  assert.match(confirmText('copy', 3, 'Shelf'), /Copy 3 books.*originals stay.*without a reading place/)
  assert.equal(doneText('move', 2, 'Shelf'), 'Moved 2 books to “Shelf”.')
})
