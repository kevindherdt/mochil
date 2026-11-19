import test from 'node:test'
import assert from 'node:assert/strict'
import { chapterSentences, sentenceSpans, splitLead } from '../static/narration-text.js'

// A document is a body with blocks that hold text nodes; that is all the extractor reads.
function documentOf(...blocks) {
  const body = { tagName: 'BODY', parentElement: null }
  const nodes = []
  for (const [tag, ...chunks] of blocks) {
    const element = { tagName: tag, parentElement: body }
    for (const chunk of chunks) {
      if (typeof chunk === 'string') nodes.push({ data: chunk, parentElement: element })
      else nodes.push({ data: chunk.text, parentElement: { tagName: chunk.tag, parentElement: element } })
    }
  }
  return { body, createTreeWalker() { let at = 0; return { nextNode: () => nodes[at++] ?? null } } }
}

const spans = text => sentenceSpans(text).map(([from, to]) => text.slice(from, to).trim())

test('splits sentences at their ends, with closing quotes kept', () => {
  assert.deepEqual(spans('He ran home quickly. “Where were you?” she asked him. Nobody answered at all.'),
    ['He ran home quickly.', '“Where were you?” she asked him.', 'Nobody answered at all.'])
})

test('abbreviations and initials do not end a sentence', () => {
  assert.deepEqual(spans('Dr. Smith met J.R. Tolkien at the old station. They talked for hours.'),
    ['Dr. Smith met J.R. Tolkien at the old station.', 'They talked for hours.'])
})

test('a short fragment is read together with the next sentence', () => {
  assert.deepEqual(spans('Hello! I came back to see you today.'), ['Hello! I came back to see you today.'])
})

test('a very long sentence is read in parts at commas', () => {
  const long = Array.from({ length: 30 }, (_, i) => `a clause number ${i} of the story`).join(', ') + '.'
  const parts = spans(long)
  assert.ok(parts.length > 1)
  assert.ok(parts.every(part => part.length <= 420), parts.map(part => part.length).join())
  assert.equal(parts.join(' ').replace(/\s+/g, ' '), long)
})

test('sentences come out in reading order and point at their text', () => {
  const doc = documentOf(['H1', 'Chapter One'], ['P', 'The lantern swung over the harbour. ', { tag: 'EM', text: 'It was late' }, ' and the sea was calm.'])
  const found = chapterSentences(doc)
  assert.deepEqual(found.map(s => s.text), ['Chapter One', 'The lantern swung over the harbour.', 'It was late and the sea was calm.'])
  const second = found[1]
  assert.equal(second.startOffset, 0)
  assert.equal(second.endNode.data.slice(0, second.endOffset), 'The lantern swung over the harbour.')
  const third = found[2]
  assert.equal(third.startNode.data, 'It was late', 'a sentence can start inside inline markup')
  assert.equal(third.endNode.data.slice(0, third.endOffset), ' and the sea was calm.')
})

test('text in scripts, styles and ruby annotations is not read', () => {
  const doc = documentOf(['P', 'Visible words are here today.', { tag: 'RT', text: 'furigana' }], ['P', { tag: 'STYLE', text: 'p { color: red }' }])
  assert.deepEqual(chapterSentences(doc).map(s => s.text), ['Visible words are here today.'])
})

test('an empty or break-only chapter gives nothing to read', () => {
  assert.deepEqual(chapterSentences(documentOf(['P', '* * *'], ['P', '   '])), [])
  assert.deepEqual(chapterSentences(null), [])
})

test('boilerplate sentences are left out of what is read', () => {
  const doc = documentOf(['P', 'Copyright © Wilbur Smith 2014'], ['P', 'The lantern swung over the harbour.'], ['P', 'Visit www.example.com for more.'])
  assert.deepEqual(chapterSentences(doc).map(s => s.text), ['The lantern swung over the harbour.'])
})

test('a long sentence knows where its opening clause ends, and splits there', () => {
  const doc = documentOf(['P', 'When the old captain finally came ashore at dusk, he counted every rope twice before he spoke.'], ['P', 'Short one, really.'])
  const [long, short] = chapterSentences(doc)
  assert.equal(long.lead.text, 'When the old captain finally came ashore at dusk,')
  assert.equal(short.lead, undefined, 'a short sentence is not cut')
  const [first, rest] = splitLead(long)
  assert.equal(first.text, 'When the old captain finally came ashore at dusk,')
  assert.equal(rest.text, 'he counted every rope twice before he spoke.')
  assert.equal(first.startNode, long.startNode)
  assert.equal(first.endNode, long.lead.endNode)
  assert.equal(rest.startOffset, long.lead.endOffset, 'the second part starts where the first ends')
  assert.equal(rest.endOffset, long.endOffset)
  assert.deepEqual(splitLead(short), [short])
})
