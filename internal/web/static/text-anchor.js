// Text anchors: a quote of the text that starts a page, with context, so any
// reader (web, Android, KOReader) can find the same place in the same book.

// A text anchor is a short quote of what starts the visible page with a little
// context on both sides, so any reader can find the same place again.
export function anchorAroundTextNode(node, offset, spineIndex, progress) {
  const walker = node.ownerDocument.createTreeWalker(node.ownerDocument.body, NodeFilter.SHOW_TEXT)
  walker.currentNode = node
  let before = node.data.slice(0, Math.min(offset, node.data.length))
  let previous = node
  while (before.length < 1200 && (previous = walker.previousNode())) before = `${previous.data} ${before}`
  walker.currentNode = node
  let after = node.data.slice(Math.min(offset, node.data.length))
  let following = node
  while (after.length < 1800 && (following = walker.nextNode())) after += ` ${following.data}`
  const words = value => value.replace(/\u00ad/g, '').replace(/\s+/g, ' ').trim().split(' ').filter(Boolean)
  const preceding = words(before)
  const upcoming = words(after)
  const exact = upcoming.slice(0, 16).join(' ')
  if (exact.length < 24) return null
  return {
    version: 2,
    prefix: preceding.slice(-24).join(' '),
    exact,
    suffix: upcoming.slice(16, 48).join(' '),
    spine_index: spineIndex,
    progress,
  }
}

// Text anchors are compared with every kind of whitespace removed, so quotes
// taken from a document where blocks and inline elements are separated
// differently (another reader, or minified XHTML) still match.
export function canonicalText(value) {
  return String(value || '').normalize('NFKC').toLocaleLowerCase()
    .replace(/\u00ad/g, '').replace(/[‘’]/g, "'").replace(/[‐‑‒–—―]/g, '-')
    .replace(/\s+/gu, '')
}

export function searchableDocument(doc) {
  const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT)
  const points = []
  let normalized = ''
  let node
  while ((node = walker.nextNode())) {
    for (let offset = 0; offset < node.data.length;) {
      const sourceCharacter = String.fromCodePoint(node.data.codePointAt(offset))
      const chunk = canonicalText(sourceCharacter)
      for (const character of chunk) {
        normalized += character
        points.push({ node, offset })
      }
      offset += sourceCharacter.length
    }
  }
  return { normalized, points }
}
