import test from 'node:test'
import assert from 'node:assert/strict'

globalThis.NodeFilter = { SHOW_TEXT: 4 }
const { findInDocument, createReadiumSearch } = await import('../static/readium-search.js')

// A document is only text nodes to the search, so a list of them is enough.
const documentOf = (...chunks) => {
  const nodes = chunks.map(data => ({ data, length: data.length }))
  return {
    body: { textContent: chunks.join('') },
    createTreeWalker() { let at = 0; return { nextNode: () => nodes[at++] ?? null } },
  }
}

test('finds a phrase across spacing, case and node boundaries, with its surroundings', () => {
  const doc = documentOf('He said that the ', 'Experience  Points ', 'were not refundable at all.')
  const [hit] = findInDocument(doc, 'experience points')
  assert.equal(hit.excerpt.match, 'Experience  Points'.replace(/\s+/g, ' '))
  assert.match(hit.excerpt.pre, /that the $/)
  assert.match(hit.excerpt.post, /^ were not/)
  assert.ok(hit.ratio > 0.2 && hit.ratio < 0.5)
})

test('preserves spaces around highlighted search matches without adding spaces inside words', () => {
  const [whole] = findInDocument(documentOf('Test book for isolated layout audit.'), 'isolated')
  assert.equal(whole.excerpt.pre + whole.excerpt.match + whole.excerpt.post, 'Test book for isolated layout audit.')
  const [partial] = findInDocument(documentOf('an isolation test'), 'isol')
  assert.equal(partial.excerpt.pre + partial.excerpt.match + partial.excerpt.post, 'an isolation test')
})

test('ignores curly quotes and dashes the way a reader types them', () => {
  const doc = documentOf('It wasn’t as if heroes — well, any — existed.')
  assert.equal(findInDocument(doc, "wasn't").length, 1)
  assert.equal(findInDocument(doc, 'any - existed').length, 1)
  assert.equal(findInDocument(doc, 'nothing like it').length, 0)
})

test('caps hits per chapter and finds each occurrence once', () => {
  const doc = documentOf('tip '.repeat(100))
  assert.equal(findInDocument(doc, 'tip', 5).length, 5)
  assert.equal(findInDocument(documentOf('aaaa'), 'aa').length, 2, 'matches do not overlap')
})

test('searching a book reports progress and only chapters that match, skipping generated pages', async () => {
  const chapters = { 'a.xhtml': documentOf('nothing here'), 'b.xhtml': documentOf('the needle is here'), '_bobo/cover.xhtml': documentOf('needle') }
  const source = createReadiumSearch({
    order: Object.keys(chapters).map(href => ({ href, type: 'application/xhtml+xml' })),
    titleFor: href => ({ 'b.xhtml': 'Chapter B' })[href],
    fetchDocument: async href => chapters[href],
  })
  const seen = []
  for await (const result of source.search('needle')) seen.push(result)
  const groups = seen.filter(result => result.subitems)
  assert.equal(groups.length, 1)
  assert.equal(groups[0].label, 'Chapter B')
  assert.equal(groups[0].subitems[0].target.href, 'b.xhtml')
  assert.equal(seen.filter(result => Number.isFinite(result.progress)).length, 2)
  assert.equal(seen.at(-1), 'done')
})

test('a chapter that cannot be fetched does not stop the search', async () => {
  const source = createReadiumSearch({
    order: [{ href: 'bad.xhtml' }, { href: 'good.xhtml' }],
    titleFor: () => '',
    fetchDocument: async href => { if (href === 'bad.xhtml') throw new Error('500'); return documentOf('a needle') },
  })
  const seen = []
  for await (const result of source.search('needle')) seen.push(result)
  assert.equal(seen.filter(result => result.subitems).length, 1)
})
