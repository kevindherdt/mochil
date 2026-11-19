import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { createCollections } from '../static/collections.js'
import { createAppViewController } from '../static/app-view-controller.js'

// Exercise the actual isActive wiring from app.js with the real controllers.
const appSource = readFileSync(new URL('../static/app.js', import.meta.url), 'utf8')
const activeExpression = appSource.match(/isActive:\s*\(\)\s*=>\s*([^,]+),/)?.[1]
assert.ok(activeExpression, 'app.js must wire collections isActive')
const appIsActive = new Function('collectionMode', '$', `return (${activeExpression})`)
const tick = () => new Promise(resolve => setImmediate(resolve))

function fixture() {
  const nodes = new Map()
  const node = selector => {
    if (!nodes.has(selector)) nodes.set(selector, {
      dataset: {}, textContent: '', innerHTML: '', value: '', hidden: false,
      classList: { toggle() {} }, setAttribute() {}, querySelectorAll: () => [],
      showModal() { this.opened = true },
    })
    return nodes.get(selector)
  }
  const requests = [], events = []
  const button = { dataset: { id: 'a' } }
  let collectionMode = true
  const collections = createCollections({
    document: { querySelector: () => ({}), defaultView: { dispatchEvent: event => events.push(event.type) } },
    $: node, $$: selector => selector.includes('.collection-grid') ? [button] : [],
    escapeHTML: String, requestJSON: url => new Promise((resolve, reject) => requests.push({ url, resolve, reject })),
    bookMarkup: book => `<article>${book.title}</article>`, installCoverFallbacks() {},
    getSelectedLibrary: () => 'books', getItemLayout: () => 'covers',
    isActive: () => appIsActive(collectionMode, node), abortBrowse() {},
  })
  const views = createAppViewController({
    $: node, mediaNavigator: { hide() {} }, searchController: { setOpen() {} }, searchToggle: {},
    hasLibraries: () => true, openSettings() {}, loadLibrary() {}, loadDashboard() {},
    updateLayout() {}, requestFrame() {},
  })
  return { node, requests, events, collections, views, button }
}

test('collection dialog remains usable when the library view is hidden', async () => {
  const f = fixture()
  f.views.show('settings')
  const opening = f.collections.openFor([{ type: 'book', library_id: 'books', book_id: 'x' }])
  assert.equal(f.node('#collection-dialog').opened, true)
  f.requests[0].resolve({ data: [{ id: 'a', name: 'Alpha', count: 1, members: [] }] })
  await opening
  assert.match(f.node('#collection-choices').innerHTML, /Alpha/)
  assert.equal(f.node('#collection-dialog-status').textContent, '')
})

for (const view of ['settings', 'dashboard']) {
  for (const phase of ['overview', 'detail']) {
    for (const outcome of ['success', 'failure']) {
      test(`leaving library for ${view} discards pending ${phase} ${outcome}`, async () => {
        const f = fixture()
        f.views.show('library')
        let pending = f.collections.load()
        if (phase === 'detail') {
          f.requests[0].resolve({ data: [{ id: 'a', name: 'Alpha', count: 1, members: [] }] })
          await pending
          f.button.onclick()
        }
        const request = f.requests.at(-1)
        f.views.show(view)
        assert.equal(f.node('#library-view').hidden, true)
        assert.equal(f.node('#collection-index').hidden, false)
        f.node('#status').textContent = 'New view status'
        const markup = f.node('#collection-index').innerHTML
        const events = f.events.length
        if (outcome === 'success') request.resolve({ data: phase === 'overview' ? [{ id: 'stale', name: 'Stale', count: 0, members: [] }] : [{ title: 'Stale book' }] })
        else request.reject(new Error('Stale failure'))
        await tick()
        assert.equal(f.node('#status').textContent, 'New view status')
        assert.equal(f.node('#collection-index').innerHTML, markup)
        assert.equal(f.events.length, events)
      })
    }
  }
}
