import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.location = { origin: 'https://books.example' }
let nextURL = 0
const revoked = []
const blobs = []
globalThis.URL.createObjectURL = blob => {
  blobs.push(blob)
  return `blob:comic-${++nextURL}`
}
globalThis.URL.revokeObjectURL = value => revoked.push(value)

const { makeStreamedComicBook } = await import('../static/streamed-comic-book.js')

test('streamed comic creates bounded lazy fixed-layout sections', async () => {
  const book = makeStreamedComicBook({ title: 'Manga', pageCount: 3, pageURL: '/stream/manga/id?page={pageNumber}' })
  assert.equal(book.sections.length, 3)
  assert.equal(book.rendition.layout, 'pre-paginated')
  assert.equal(await book.sections[0].load(), 'blob:comic-1')
  assert.equal(await book.sections[0].load(), 'blob:comic-1')
  const markup = await blobs[0].text()
  assert.doesNotMatch(markup, /name="viewport"/)
  assert.match(markup, /<img[^>]+\/stream\/manga\/id\?page=0/)
  book.sections[0].unload()
  assert.deepEqual(revoked, ['blob:comic-1'])
})

test('streamed comic rejects missing templates and unreasonable page counts', () => {
  assert.throws(() => makeStreamedComicBook({ pageCount: 0, pageURL: '/page/{pageNumber}' }), /invalid page count/)
  assert.throws(() => makeStreamedComicBook({ pageCount: 20_001, pageURL: '/page/{pageNumber}' }), /invalid page count/)
  assert.throws(() => makeStreamedComicBook({ pageCount: 2, pageURL: '/page/1' }), /does not provide page streaming/)
  const remote = makeStreamedComicBook({ pageCount: 2, pageURL: 'https://evil.example/page/{pageNumber}' })
  assert.throws(() => remote.sections[0].load(), /must come from this Mochil server/)
})
