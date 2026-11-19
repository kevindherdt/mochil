import assert from 'node:assert/strict'
import test from 'node:test'
import { createReaderPositionHistory } from '../static/reader-position-history.js'

const response = (status, body) => ({ ok: status >= 200 && status < 300, json: async () => body })

function fixture(fetchImpl) {
  const elements = new Map()
  const element = id => elements.get(id) || elements.set(id, {
    innerHTML: '', open: false, showModal() { this.open = true }, close() { this.open = false }, append() {},
  }).get(id)
  const history = createReaderPositionHistory({
    $: selector => element(selector), endpoint: '/api/reading/books/one', deviceID: 'device-one',
    readingSync: { revision: 4, applyingRemote: false, acceptRemote() {} },
    goToPosition: async () => {}, currentPosition: () => null, renderLocation() {}, fetchImpl,
    documentRef: { createElement: () => ({ textContent: '', get innerHTML() { return this.textContent.replaceAll('&', '&amp;').replaceAll('<', '&lt;') } }) },
  })
  return { history, element }
}

test('position history renders a bounded empty state', async () => {
  const { history, element } = fixture(async url => {
    assert.equal(url, '/api/reading/books/one/history')
    return response(200, [])
  })
  await history.open()
  assert.equal(element('#position-history').open, true)
  assert.match(element('#history-list').innerHTML, /No earlier positions/)
})

test('position history turns unreadable responses into recoverable UI', async () => {
  const { history, element } = fixture(async () => response(503, null))
  await history.open()
  assert.match(element('#history-list').innerHTML, /History is unavailable/)
})
