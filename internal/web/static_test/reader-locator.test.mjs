import assert from 'node:assert/strict'
import test from 'node:test'
import { createReaderLocator } from '../static/reader-locator.js'

function publicationView() {
  const text = { nodeType: 3, data: 'Chapter opening text' }
  const paragraph = { localName: 'p', children: [], childNodes: [text] }
  const body = { children: [paragraph], childNodes: [paragraph] }
  const range = {
    setStart(node, offset) { this.node = node; this.offset = offset },
    collapse(value) { this.collapsed = value },
  }
  const doc = { body, createRange: () => range }
  return {
    view: {
      book: { sections: [{ createDocument: async () => doc }] },
      getCFI: (index, selectedRange) => selectedRange
        ? `exact:${index}:${selectedRange.offset}`
        : `chapter:${index}`,
    },
  }
}

test('KOReader XPointers resolve to an exact Foliate CFI when the DOM path matches', async t => {
  const previousNode = globalThis.Node
  globalThis.Node = { TEXT_NODE: 3 }
  t.after(() => { globalThis.Node = previousNode })
  const { view } = publicationView()
  const locator = createReaderLocator(view)
  const resolved = await locator.resolveKoReaderPosition('/body/DocFragment[1]/body/p[1]/text().7')

  assert.deepEqual(resolved, { cfi: 'exact:0:7', accuracy: 'exact' })
})

test('a valid spine with wrapper drift falls back to its chapter instead of guessing', async t => {
  const previousNode = globalThis.Node
  globalThis.Node = { TEXT_NODE: 3 }
  t.after(() => { globalThis.Node = previousNode })
  const { view } = publicationView()
  const locator = createReaderLocator(view)
  const resolved = await locator.resolveKoReaderPosition('/body/DocFragment[1]/body/section[2]/p[1]')

  assert.deepEqual(resolved, { cfi: 'chapter:0', accuracy: 'chapter' })
})

test('malformed and out-of-range XPointers are rejected', async () => {
  const { view } = publicationView()
  const locator = createReaderLocator(view)

  assert.equal(await locator.resolveKoReaderPosition('not-an-xpointer'), null)
  assert.equal(await locator.resolveKoReaderPosition('/body/DocFragment[4]/body/p[1]'), null)
})

test('web anchors require an actual rendered text range', () => {
  const locator = createReaderLocator({ renderer: { getContents: () => [] } })
  assert.equal(locator.webAnchorFromLocation({ fraction: 0.4 }), null)
})
