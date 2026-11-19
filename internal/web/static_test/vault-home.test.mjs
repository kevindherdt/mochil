import test from 'node:test'
import assert from 'node:assert/strict'
import { recentPages, sinceText, vaultHomeHTML } from '../static/vault-home.js'

const esc = text => String(text)
const now = Date.parse('2026-10-31T12:00:00Z')
const pages = [
  { id: 'a', path: 'Work/old.md', title: 'Old', format: 'md', modified: '2026-10-01T00:00:00Z' },
  { id: 'b', path: 'new.md', title: 'New', format: 'md', modified: '2026-10-31T08:00:00Z' },
  { id: 'c', path: 'pic.png', title: 'Pic', format: 'png', modified: '2026-10-31T09:00:00Z' },
  { kind: 'folder', path: 'Work' },
]

test('recent pages are the newest markdown pages first', () => {
  assert.deepEqual(recentPages(pages).map(page => page.id), ['b', 'a'])
})

test('dates read as words for the last week', () => {
  assert.equal(sinceText('2026-10-31T01:00:00Z', now), 'Today')
  assert.equal(sinceText('2026-10-30T01:00:00Z', now), 'Yesterday')
  assert.equal(sinceText('2026-10-28T01:00:00Z', now), '3 days ago')
})

test('the home offers actions and recent pages; read-only vaults get no write actions; an empty vault keeps the message', () => {
  const html = vaultHomeHTML(pages, { message: 'x', writable: true, escapeHTML: esc, now })
  assert.match(html, /data-vault-create="page"/)
  assert.match(html, /data-vault-daily/)
  assert.match(html, /data-document="b"/)
  const readOnly = vaultHomeHTML(pages, { message: 'x', writable: false, escapeHTML: esc, now })
  assert.doesNotMatch(readOnly, /data-vault-create|data-vault-daily/)
  assert.equal(vaultHomeHTML([], { message: 'No pages yet.', writable: true, escapeHTML: esc }), '<p class="vault-empty">No pages yet.</p>')
})

test('the tag row lists the most used tags first and links each to its page', async () => {
  const { tagRowHTML } = await import('../static/vault-home.js')
  const html = tagRowHTML([{ tag: 'b', ids: ['1'] }, { tag: 'a', ids: ['1', '2'] }], esc)
  assert.ok(html.indexOf('#a') < html.indexOf('#b'))
  assert.match(html, /href="#tag:a"/)
  assert.equal(tagRowHTML([], esc), '')
})

test('a property link carries its key and value and reads back', async () => {
  const { propertyHref, parsePropertyHref } = await import('../static/vault-properties.js')
  const href = propertyHref('Status', 'In progress = 2')
  assert.deepEqual(parsePropertyHref(href), { key: 'status', value: 'In progress = 2' })
})

test('properties are written back as simple YAML and the body is never touched', async () => {
  const { propertiesToYAML, withProperties, editableRows } = await import('../static/vault-properties-edit.js')
  const note = '---\nstatus: draft\naliases:\n  - One\n---\n# Title\n\nBody: stays #as is\n'
  const rows = editableRows(note)
  assert.equal(propertiesToYAML(rows), 'status: draft\naliases:\n  - One\n')
  rows[0].values = ['in: progress']
  assert.equal(withProperties(note, rows), '---\nstatus: "in: progress"\naliases:\n  - One\n---\n# Title\n\nBody: stays #as is\n')
  assert.equal(withProperties(note, []), '# Title\n\nBody: stays #as is\n')
  assert.equal(editableRows('# no header'), null)
})
