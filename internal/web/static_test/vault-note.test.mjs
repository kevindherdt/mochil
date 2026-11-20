import assert from 'node:assert/strict'
import test from 'node:test'
import { dropRepeatedTitle, vaultNoteHTML } from '../static/vault-note.js'

const escapeHTML = value => String(value).replace(/[&<>"']/g, character => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character])
const item = { title: 'Plan', path: 'Projects/Deep/Plan.md', download_url: '/x?download=1' }

test('a note shows where it lives and what can be done with it', () => {
  const html = vaultNoteHTML({ item, libraryName: 'Notes', content: '<p>Hi</p>', canEdit: true, escapeHTML })
  assert.match(html, /<span>Notes<\/span><i[^>]*>\/<\/i><span>Projects<\/span><i[^>]*>\/<\/i><span>Deep<\/span>/)
  assert.doesNotMatch(html, /<span>Plan\.md<\/span>/)
  assert.match(html, /data-vault-edit/)
  assert.match(html, /<h1 class="vault-title">Plan<\/h1><article><p>Hi<\/p><\/article>/)
})

test('read-only notes offer no edit button and names are escaped', () => {
  const html = vaultNoteHTML({ item: { ...item, title: '<b>', path: 'A&B/<b>.md' }, content: '', canEdit: false, escapeHTML })
  assert.doesNotMatch(html, /data-vault-edit/)
  assert.match(html, /A&amp;B/)
  assert.doesNotMatch(html, /<b>/)
})

test('a leading heading that repeats the title is dropped, other headings stay', () => {
  const heading = { tagName: 'H1', textContent: '  plan ', removed: false, remove() { this.removed = true } }
  dropRepeatedTitle({ querySelector: () => ({ firstElementChild: heading }) }, 'Plan')
  assert.equal(heading.removed, true)
  const other = { tagName: 'H1', textContent: 'Something else', remove() { throw new Error('kept') } }
  dropRepeatedTitle({ querySelector: () => ({ firstElementChild: other }) }, 'Plan')
  const section = { tagName: 'H2', textContent: 'Plan', remove() { throw new Error('kept') } }
  dropRepeatedTitle({ querySelector: () => ({ firstElementChild: section }) }, 'Plan')
})
