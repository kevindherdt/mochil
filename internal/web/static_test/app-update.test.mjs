import test from 'node:test'
import assert from 'node:assert/strict'
import { watchForUpdates } from '../static/app-update.js'

function fakeWindow(scriptSrc, serverHtml) {
  const appended = []
  const listeners = {}
  const document = {
    visibilityState: 'visible',
    body: { append: node => appended.push(node) },
    querySelectorAll: () => scriptSrc ? [{ src: `http://app.test${scriptSrc}` }] : [],
    createElement: () => ({ style: {}, append() {}, setAttribute() {} }),
    addEventListener: (name, fn) => { listeners[name] = fn },
    removeEventListener() {},
  }
  return {
    appended, listeners,
    win: {
      document, location: { href: 'http://app.test/', reload() {} },
      fetch: async () => ({ ok: true, text: async () => serverHtml }),
      setInterval: () => 1, clearInterval() {},
    },
  }
}
const page = hash => `<script type="module" src="/assets/dist/app-${hash}.js?x=1"></script>`

test('a different bundle name on the server announces an update, once', async () => {
  const { win, appended, listeners } = fakeWindow('/assets/dist/app-OLD.js', page('NEW'))
  watchForUpdates(win)
  await listeners.visibilitychange()
  await listeners.visibilitychange()
  assert.equal(appended.length, 1)
})

test('the same bundle, unbundled sources and failures stay quiet', async () => {
  const same = fakeWindow('/assets/dist/app-OLD.js', page('OLD'))
  watchForUpdates(same.win)
  await same.listeners.visibilitychange()
  assert.equal(same.appended.length, 0)

  const dev = fakeWindow('/assets/app.js', page('NEW'))
  watchForUpdates(dev.win)
  assert.equal(dev.listeners.visibilitychange, undefined)

  const down = fakeWindow('/assets/dist/app-OLD.js', page('NEW'))
  down.win.fetch = async () => { throw new Error('offline') }
  watchForUpdates(down.win)
  await down.listeners.visibilitychange()
  assert.equal(down.appended.length, 0)
})
