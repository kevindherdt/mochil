import test from 'node:test'
import assert from 'node:assert/strict'
import { createBookStatus, currentStatus, statusMenuHTML } from '../static/book-status.js'

const esc = text => String(text)

test('the current status is read from the book', () => {
  assert.equal(currentStatus({}), 'unread')
  assert.equal(currentStatus({ reading_progress: { fraction: 0.3 } }), 'reading')
  assert.equal(currentStatus({ reading_progress: { fraction: 1, finished: true } }), 'read')
  assert.equal(currentStatus({ reading_progress: { fraction: 0.3, status: 'dropped' } }), 'dropped')
})

test('the menu marks the current status and offers all five', () => {
  const html = statusMenuHTML({ title: 'Dune', reading_progress: { status: 'on_hold' } }, esc)
  assert.equal([...html.matchAll(/data-reading-status=/g)].length, 5)
  assert.match(html, /aria-checked="true" data-reading-status="on_hold"/)
  assert.match(html, /aria-checked="false" data-reading-status="read"/)
})

test('choosing a status posts it for that book and reloads on success', async () => {
  let handler
  const document = { addEventListener: (_, fn) => { handler = fn } }
  const calls = []
  let reloads = 0
  createBookStatus({ document, reload: () => reloads++, fetchImpl: async (url, init) => { calls.push([url, JSON.parse(init.body)]); return { ok: true } } })
  const menu = { removeAttribute: name => calls.push(['closed', name]) }
  const button = { dataset: { readingStatus: 'dropped' }, closest: selector => selector === '.book-more' ? menu : { dataset: { library: 'books', book: 'a b' } } }
  await handler({ target: { closest: () => button } })
  assert.deepEqual(calls[0], ['/api/reading/books/a%20b/status', { status: 'dropped' }])
  assert.equal(reloads, 1)
})
