// Searching a book for the Readium reader. Readium shows one chapter at a time
// and has no search of its own, so this reads each chapter's text from the same
// URLs the reader uses, finds the words ignoring case, spacing and typographic
// punctuation, and says where in the chapter each hit sits.
import { canonicalText, searchableDocument } from './text-anchor.js?v=1'

const CONTEXT = 50
const PER_CHAPTER = 40

const squash = value => value.replace(/\s+/gu, ' ').trim()

/** Every place `query` occurs in a chapter, with the words around it. Pure over a parsed document. */
export function findInDocument(doc, query, limit = PER_CHAPTER) {
  const needle = canonicalText(query)
  if (!needle || !doc?.body) return []
  // Cheap test first: most chapters have no hit, and mapping every character is the costly part.
  if (!canonicalText(doc.body.textContent).includes(needle)) return []
  const { normalized, points } = searchableDocument(doc)
  const starts = new Map()
  let plain = ''
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
  // Keep inline text contiguous, but restore a word boundary between HTML blocks.
  const blocks = 'address,article,aside,blockquote,dd,div,dl,dt,figcaption,figure,footer,h1,h2,h3,h4,h5,h6,header,li,main,nav,ol,p,pre,section,table,td,th,tr,ul'
  let previousBlock = null
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const block = node.parentElement?.closest(blocks) || doc.body
    if (plain && block !== previousBlock && !/\s$/u.test(plain) && !/^\s/u.test(node.data)) plain += ' '
    starts.set(node, plain.length)
    plain += node.data
    previousBlock = block
  }
  const plainAt = index => starts.get(points[index].node) + points[index].offset
  const hits = []
  for (let at = normalized.indexOf(needle); at >= 0 && hits.length < limit; at = normalized.indexOf(needle, at + needle.length)) {
    const from = plainAt(at)
    const last = plainAt(at + needle.length - 1)
    const to = Math.min(plain.length, last + 1)
    hits.push({
      ratio: at / Math.max(1, normalized.length),
      excerpt: { pre: plain.slice(Math.max(0, from - CONTEXT), from).replace(/\s+/gu, ' ').trimStart().replace(/^/, from > CONTEXT ? '…' : ''), match: squash(plain.slice(from, to)), post: plain.slice(to, to + CONTEXT).replace(/\s+/gu, ' ').trimEnd() + (to + CONTEXT < plain.length ? '…' : '') },
    })
  }
  return hits
}

/** Source for the search dialog: walks the reading order one chapter at a time. */
export function createReadiumSearch({ order, titleFor, fetchDocument, chapterWindow = () => null }) {
  const readable = order.filter(link => !link.href.startsWith('_bobo/'))
  async function * search(query) {
    for (const [index, link] of readable.entries()) {
      let doc
      try { doc = await fetchDocument(link.href) } catch { doc = null }
      yield { progress: (index + 1) / readable.length }
      const hits = doc ? findInDocument(doc, query) : []
      if (hits.length) {
        yield { label: titleFor(link.href) || `Chapter ${index + 1}`, subitems: hits.map(hit => ({ excerpt: hit.excerpt, target: { href: link.href, type: link.type, ratio: hit.ratio, query } })) }
      }
    }
    yield 'done'
  }
  return { search, clear() { clearHighlight(chapterWindow()) } }
}

/** Marks a hit in the chapter that is showing, using the browser's highlight API where it exists. */
export function highlightNear(frameWindow, query, ratio) {
  const doc = frameWindow?.document
  const needle = canonicalText(query)
  if (!doc?.body || !needle || !frameWindow.CSS?.highlights || typeof frameWindow.Highlight !== 'function') return null
  const { normalized, points } = searchableDocument(doc)
  let best = -1
  for (let at = normalized.indexOf(needle); at >= 0; at = normalized.indexOf(needle, at + needle.length)) {
    if (best < 0 || Math.abs(at / normalized.length - ratio) < Math.abs(best / normalized.length - ratio)) best = at
  }
  if (best < 0) return null
  const start = points[best], end = points[best + needle.length - 1]
  const range = doc.createRange()
  range.setStart(start.node, start.offset)
  range.setEnd(end.node, Math.min(end.node.length, end.offset + 1))
  if (!doc.getElementById('bobo-search-style')) {
    const style = doc.createElement('style')
    style.id = 'bobo-search-style'
    style.textContent = '::highlight(bobo-search){background-color:#f2c14e;color:#241f1a}'
    doc.head.append(style)
  }
  frameWindow.CSS.highlights.set('bobo-search', new frameWindow.Highlight(range))
  return range
}

export function clearHighlight(frameWindow) {
  try { frameWindow?.CSS?.highlights?.delete('bobo-search') } catch {}
}
