// Finds the sentences of a chapter, in reading order, with where each one sits in
// the page so it can be spoken, marked and scrolled to. It only reads text nodes
// and their parents, so it can be tested without a browser.

import { worthReading } from './narration-clean.js?v=1'

const BLOCKS = new Set(['P', 'DIV', 'LI', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'BLOCKQUOTE', 'TD', 'TH', 'DT', 'DD', 'SECTION', 'ARTICLE', 'FIGCAPTION', 'PRE', 'BODY'])
const SKIPPED = new Set(['SCRIPT', 'STYLE', 'RT', 'RP', 'NOSCRIPT', 'SVG', 'MATH'])
const ABBREVIATION = /\b(?:Mr|Mrs|Ms|Dr|Prof|St|Sr|Jr|vs|etc|e\.g|i\.e|dhr|mevr)$/i
const INITIALS = /(?:\b[A-Za-z]\.)+[A-Za-z]$/
const SENTENCE_END = /[.!?…]["'”’»)\]]*(?:\s+|$)/g
const MIN_WORDS = 3
const MAX_CHARS = 400

// Where a long sentence can be cut so its opening clause is spoken first: after a comma, semicolon or colon.
const LEAD = /[,;:]\s/g
const LEAD_MIN = 25, LEAD_MAX = 90, REST_MIN = 30

function leadEnd(raw) {
  LEAD.lastIndex = LEAD_MIN
  const match = LEAD.exec(raw)
  return match && match.index < LEAD_MAX && raw.length - match.index > REST_MIN ? match.index + 1 : 0
}

const wordCount = text => text.trim().split(/\s+/).filter(Boolean).length

function blockOf(node) {
  for (let parent = node.parentElement; parent; parent = parent.parentElement) {
    if (BLOCKS.has(parent.tagName)) return parent
  }
  return null
}

function skipped(node) {
  for (let parent = node.parentElement; parent; parent = parent.parentElement) {
    if (SKIPPED.has(String(parent.tagName).toUpperCase())) return true
  }
  return false
}

/** Cuts a block's text into [start, end) character ranges, one per sentence. */
export function sentenceSpans(text) {
  const spans = []
  let start = 0
  SENTENCE_END.lastIndex = 0
  for (let match = SENTENCE_END.exec(text); match; match = SENTENCE_END.exec(text)) {
    if (text[match.index] === '.') {
      const before = text.slice(0, match.index)
      if (ABBREVIATION.test(before) || INITIALS.test(before)) continue
    }
    const end = match.index + match[0].length
    // “Where?” she asked: a lowercase word after the closing quote continues the sentence.
    if (/^\p{Ll}/u.test(text.slice(end, end + 1))) continue
    spans.push([start, end])
    start = end
  }
  if (text.slice(start).trim()) spans.push([start, text.length])
  // A greeting or an exclamation alone sounds clipped; read it with what follows.
  const merged = []
  for (const span of spans) {
    const last = merged[merged.length - 1]
    if (last && wordCount(text.slice(last[0], last[1])) < MIN_WORDS) last[1] = span[1]
    else merged.push([...span])
  }
  // A very long sentence is read in parts, at commas, so audio starts sooner and stays within limits.
  const parts = []
  for (const [from, to] of merged) {
    let cursor = from
    while (to - cursor > MAX_CHARS) {
      const window = text.slice(cursor, cursor + MAX_CHARS)
      const cut = Math.max(window.lastIndexOf(', '), window.lastIndexOf('; '), window.lastIndexOf(': '))
      const end = cut > MAX_CHARS / 3 ? cursor + cut + 2 : cursor + (window.lastIndexOf(' ') > 0 ? window.lastIndexOf(' ') + 1 : MAX_CHARS)
      parts.push([cursor, end])
      cursor = end
    }
    parts.push([cursor, to])
  }
  return parts.filter(([from, to]) => /\w/.test(text.slice(from, to)))
}

/**
 * @returns {{text: string, startNode: Node, startOffset: number, endNode: Node, endOffset: number}[]}
 */
export function chapterSentences(doc) {
  if (!doc?.body) return []
  const walker = doc.createTreeWalker(doc.body, 4 /* NodeFilter.SHOW_TEXT */)
  const blocks = []
  let current = null
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    if (!node.data || skipped(node)) continue
    const block = blockOf(node)
    if (!current || current.block !== block) {
      current = { block, text: '', pieces: [] }
      blocks.push(current)
    }
    current.pieces.push({ node, from: current.text.length })
    current.text += node.data
  }
  const sentences = []
  for (const block of blocks) {
    const locate = (offset, atEnd) => {
      for (let index = block.pieces.length - 1; index >= 0; index--) {
        const piece = block.pieces[index]
        if (offset > piece.from || (offset === piece.from && !atEnd)) {
          return { node: piece.node, offset: Math.min(piece.node.data.length, offset - piece.from) }
        }
      }
      return { node: block.pieces[0].node, offset: 0 }
    }
    for (const [from, to] of sentenceSpans(block.text)) {
      // Trim the whitespace around a sentence so its highlight hugs the words.
      let start = from, end = to
      while (start < end && /\s/.test(block.text[start])) start++
      while (end > start && /\s/.test(block.text[end - 1])) end--
      const text = block.text.slice(start, end).replace(/\s+/g, ' ')
      if (!/\w/.test(text) || !worthReading(text)) continue
      const first = locate(start, false), last = locate(end, true)
      const sentence = { text, startNode: first.node, startOffset: first.offset, endNode: last.node, endOffset: last.offset }
      const cut = leadEnd(block.text.slice(start, end))
      if (cut) {
        const lead = locate(start + cut, true)
        sentence.lead = { text: block.text.slice(start, start + cut).replace(/\s+/g, ' ').trim(), endNode: lead.node, endOffset: lead.offset }
      }
      sentences.push(sentence)
    }
  }
  return sentences
}

/**
 * Cuts a sentence into its opening clause and the rest, so speech can begin as soon as the
 * clause is ready. A sentence without a clause to cut at comes back whole.
 */
export function splitLead(sentence) {
  const { lead } = sentence
  if (!lead) return [sentence]
  const rest = sentence.text.slice(lead.text.length).trim()
  if (!rest) return [sentence]
  const { lead: _, ...whole } = sentence
  return [
    { ...whole, text: lead.text, endNode: lead.endNode, endOffset: lead.endOffset },
    { ...whole, text: rest, startNode: lead.endNode, startOffset: lead.endOffset },
  ]
}
