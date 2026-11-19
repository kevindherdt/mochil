import test from 'node:test'
import assert from 'node:assert/strict'
import { rankPages, newPagePath, hasPageNamed } from '../static/vault-find.js'
import { wikiLinkMatches } from '../static/vault-link-complete.js'
import { backlinksHTML } from '../static/vault-backlinks.js'

const pages = [
  { id: '1', title: 'Home', path: 'Home.md' },
  { id: '2', title: 'Plan', path: 'Projects/Plan.md' },
  { id: '3', title: 'Plan', path: 'Archive/Plan.md' },
  { id: '4', title: 'Reading list', path: 'Books/Reading list.md' },
  { id: '5', title: 'Homebrew', path: 'Notes/Homebrew.md' },
]

test('pages rank exact, then prefix, then word start, then anywhere', () => {
  assert.deepEqual(rankPages(pages, 'home').map(p => p.id), ['1', '5'])
  assert.deepEqual(rankPages(pages, 'list').map(p => p.id), ['4'])
  assert.deepEqual(rankPages(pages, 'proj').map(p => p.id), ['2'])
  assert.deepEqual(rankPages(pages, 'rdl').map(p => p.id), ['4'])
  assert.equal(rankPages(pages, 'zzz').length, 0)
  assert.equal(rankPages(pages, '').length, 5)
})

test('new page paths are normalized and unsafe names refused', () => {
  assert.equal(newPagePath('Ideas'), 'Ideas.md')
  assert.equal(newPagePath('/Work/Ideas.md'), 'Work/Ideas.md')
  assert.equal(newPagePath('../x'), null)
  assert.equal(newPagePath('a//b'), null)
  assert.equal(newPagePath('   '), null)
  assert.ok(hasPageNamed(pages, 'home'))
  assert.ok(hasPageNamed(pages, 'Projects/Plan'))
  assert.ok(!hasPageNamed(pages, 'Projects/Other'))
})

test('[[ suggestions use the folder path only when a name is ambiguous', () => {
  const plan = wikiLinkMatches(pages, 'plan')
  assert.deepEqual(plan.map(m => m.name).sort(), ['Archive/Plan', 'Projects/Plan'])
  assert.equal(wikiLinkMatches(pages, 'reading')[0].name, 'Reading list')
  assert.equal(wikiLinkMatches(pages, 'reading')[0].detail, 'Books')
})

test('backlinks render escaped titles and nothing when empty', () => {
  const esc = value => String(value).replace(/</g, '&lt;')
  assert.equal(backlinksHTML([], esc), '')
  const html = backlinksHTML([{ id: 'a', title: '<b>', path: 'X/<b>.md' }], esc)
  assert.ok(html.includes('data-document="a"') && html.includes('&lt;b>') && !html.includes('<b>'))
  assert.ok(html.includes('Linked from'))
})

import { parseProperties } from '../static/vault-frontmatter.js'
import { movedPath } from '../static/vault-dnd.js'
import { rankTags } from '../static/vault-tags.js'
import { createVaultMarkdownRenderer } from '../static/vault-markdown.js'

test('properties parse simple front matter and decline anything fancier', () => {
  const rows = parseProperties('title: "My Plan"\nstatus: draft\ntags: [a, b]\naliases:\n  - One\n  - Two\n')
  assert.deepEqual(rows.map(r => [r.key, r.values]), [['title', ['My Plan']], ['status', ['draft']], ['tags', ['a', 'b']], ['aliases', ['One', 'Two']]])
  assert.equal(parseProperties('a:\n  b: 1\n'), null)
  assert.equal(parseProperties('text: |\n  multi\n'), null)
})

test('pages dropped on folders keep their name; the top of the vault is ""', () => {
  assert.equal(movedPath('A/B/Plan.md', 'Work'), 'Work/Plan.md')
  assert.equal(movedPath('A/Plan.md', ''), 'Plan.md')
})

test('tag search puts prefix matches and busy tags first', () => {
  const groups = [{ tag: 'old-project', ids: ['1'] }, { tag: 'project', ids: ['1', '2'] }, { tag: 'idea', ids: ['1'] }]
  assert.deepEqual(rankTags(groups, '#proj').map(g => g.tag), ['project', 'old-project'])
})

test('the page view shows properties as rows and #tags as links, never inside code', () => {
  const render = createVaultMarkdownRenderer([], 'v')
  const html = render('---\ntags: [work]\nstatus: open\n---\n# T\nText #idea and `#code` and a # heading', 'T.md')
  assert.match(html, /<dt>status<\/dt><dd><span><a class="vault-prop" href="#prop:status=open">open<\/a><\/span>/)
  assert.match(html, /href="#tag:work"/)
  assert.match(html, /href="#tag:idea"/)
  assert.doesNotMatch(html, /href="#tag:code"/)
})

import { formatDate, expandTemplate, dailyPath, templatePages, dailyTemplate } from '../static/vault-daily.js'

test('dates and template placeholders fill in', () => {
  const day = new Date(2026, 9, 27, 9, 5)
  assert.equal(dailyPath(day), 'Daily/2026-10-27.md')
  assert.equal(formatDate(day, 'dddd, MMMM DD, YYYY'), 'Tuesday, October 27, 2026')
  assert.equal(expandTemplate('# {{title}}\n{{date}} {{time}} {{date:DD/MM}} {{unknown}}', { date: day, title: 'Plan' }), '# Plan\n2026-10-27 09:05 27/10 {{unknown}}')
})

test('templates are the pages under Templates/, and Daily is the daily one', () => {
  const entries = [
    { id: '1', title: 'Daily', path: 'Templates/Daily.md', format: 'md' },
    { id: '2', title: 'Meeting', path: 'templates/Meeting.md', format: 'md' },
    { id: '3', title: 'Daily', path: 'Notes/Daily.md', format: 'md' },
  ]
  assert.deepEqual(templatePages(entries).map(e => e.id), ['1', '2'])
  assert.equal(dailyTemplate(entries).id, '1')
})
