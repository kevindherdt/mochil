import test from 'node:test'
import assert from 'node:assert/strict'
import { createVaultTabs } from '../static/vault-tabs.js'

const escapeHTML = value => String(value).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
const memory = () => { const data = new Map(); return { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v), data } }
const page = (name) => ({ path: `${name}.md`, title: name })

test('plain opens reuse the current tab; modifier opens add one; reopening switches', () => {
  const tabs = createVaultTabs({ storage: memory(), escapeHTML })
  tabs.use('v')
  tabs.visit(page('A'))
  tabs.visit(page('B'))
  assert.deepEqual(tabs.paths, ['B.md'], 'second page replaced the first in the same tab')
  tabs.intent({ ctrlKey: true })
  tabs.visit(page('C'))
  assert.deepEqual(tabs.paths, ['B.md', 'C.md'])
  tabs.visit(page('B'))
  assert.equal(tabs.active, 'B.md')
  assert.deepEqual(tabs.paths, ['B.md', 'C.md'], 'a page that is already open is switched to, not duplicated')
  tabs.intent({ button: 1 })
  tabs.visit(page('D'))
  assert.equal(tabs.paths.length, 3)
})

test('closing the active tab moves to a neighbour; closing the last leaves none', () => {
  const tabs = createVaultTabs({ storage: memory(), escapeHTML })
  tabs.use('v')
  for (const name of ['A', 'B', 'C']) { tabs.intent({ ctrlKey: true }); tabs.visit(page(name)) }
  assert.equal(tabs.close('C.md'), 'B.md')
  assert.equal(tabs.close('A.md'), 'B.md', 'closing another tab keeps the active one')
  assert.equal(tabs.close('B.md'), '')
  assert.equal(tabs.html(), '')
})

test('tabs are remembered per vault and forget pages that are gone', () => {
  const storage = memory()
  const first = createVaultTabs({ storage, escapeHTML })
  first.use('v'); first.intent({ ctrlKey: true }); first.visit(page('A')); first.intent({ ctrlKey: true }); first.visit(page('B'))
  const second = createVaultTabs({ storage, escapeHTML })
  second.use('v')
  assert.deepEqual(second.paths, ['A.md', 'B.md'])
  assert.equal(second.active, 'B.md')
  second.reconcile([page('B')])
  assert.deepEqual(second.paths, ['B.md'])
  second.use('other')
  assert.deepEqual(second.paths, [])
})

test('the strip escapes titles and marks the active tab', () => {
  const tabs = createVaultTabs({ storage: memory(), escapeHTML })
  tabs.use('v'); tabs.visit({ path: 'x.md', title: '<b>x</b>' })
  const html = tabs.html()
  assert.ok(html.includes('&lt;b&gt;x&lt;/b&gt;') && !html.includes('<b>x'))
  assert.ok(html.includes('aria-selected="true"'))
})

test('there is a limit on open tabs, and the active one is never the one dropped', () => {
  const tabs = createVaultTabs({ storage: memory(), escapeHTML })
  tabs.use('v')
  for (let i = 0; i < 15; i++) { tabs.intent({ ctrlKey: true }); tabs.visit(page(`P${i}`)) }
  assert.equal(tabs.paths.length, 12)
  assert.equal(tabs.active, 'P14.md')
  assert.ok(tabs.paths.includes('P14.md'))
})
