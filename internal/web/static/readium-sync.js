// Position sync for the Readium preview reader. It speaks the same protocol as
// every other reader: a fraction, a text anchor, and this engine's own locator
// stored under locators.readium.
import { anchorAroundTextNode, canonicalText, searchableDocument } from './text-anchor.js?v=1'

const generated = href => href.startsWith('_bobo/')
const decoded = href => { try { return decodeURIComponent(href) } catch { return href } }

// What this reader saves for the server and keeps locally.
export function positionFromLocator(locator) {
  const fraction = locator?.locations?.totalProgression
  if (!Number.isFinite(fraction)) return null
  return { fraction, cfi: '', locators: { readium: JSON.stringify(locator.serialize ? locator.serialize() : locator) } }
}

// The closest page-sized position to a whole-book fraction.
export function nearestPosition(positions, fraction) {
  let best = null
  for (const position of positions) {
    const distance = Math.abs((position.locations?.totalProgression ?? 0) - fraction)
    if (!best || distance < best.distance) best = { position, distance }
  }
  return best?.position ?? null
}

// The server keeps every reader's last locator side by side, so a locator saved
// before another reader moved on describes an older place. It is current only
// while it still agrees with the position the server reports.
const STALE_AFTER = 0.005

function parseLocator(encoded, readingOrder, fraction) {
  try {
    const locator = JSON.parse(encoded)
    const saved = Number(locator?.locations?.totalProgression)
    if (Number.isFinite(saved) && Number.isFinite(fraction) && Math.abs(saved - fraction) > STALE_AFTER) return null
    const wanted = decoded(String(locator?.href || '').split('#')[0])
    const link = readingOrder.find(item => decoded(item.href) === wanted)
    return link ? { ...locator, href: link.href, type: locator.type || link.type } : null
  } catch { return null }
}

// Turns a position saved by any reader into a place in this book. The order is
// the most exact first: this engine's own locator, then the quoted text, then
// the whole-book fraction.
export async function resolveRemotePosition({ remote, readingOrder, positions, findText }) {
  const own = remote?.locators?.readium && parseLocator(remote.locators.readium, readingOrder, Number(remote.fraction))
  if (own) return { locator: own, accuracy: 'exact' }

  const anchor = remote?.web_anchor
  const hasGeneratedPages = readingOrder.some(item => generated(item.href))
  if (anchor && !hasGeneratedPages && Number.isInteger(anchor.spine_index) && typeof findText === 'function') {
    const preferred = anchor.spine_index
    const order = [preferred, preferred - 1, preferred + 1, ...readingOrder.map((_, index) => index)]
      .filter((index, at, all) => index >= 0 && index < readingOrder.length && all.indexOf(index) === at)
    for (const index of order) {
      const ratio = await findText(readingOrder[index].href, anchor).catch(() => null)
      if (ratio == null) continue
      const link = readingOrder[index]
      const inChapter = positions.filter(position => position.href === link.href)
      const total = inChapter[Math.min(inChapter.length - 1, Math.round(ratio * (inChapter.length - 1)))]?.locations?.totalProgression
      return {
        locator: { href: link.href, type: link.type, locations: { progression: ratio, ...(Number.isFinite(total) ? { totalProgression: total } : {}) } },
        accuracy: 'text',
      }
    }
  }

  const nearest = Number.isFinite(Number(remote?.fraction)) ? nearestPosition(positions, Number(remote.fraction)) : null
  return nearest ? { locator: nearest, accuracy: 'approximate' } : null
}

// Where the quoted text sits in a chapter, as a 0..1 share of its characters.
export function findTextRatio(doc, anchor) {
  const exact = canonicalText(anchor?.exact || anchor?.text)
  if (exact.length < 24 || !doc?.body) return null
  const searchable = searchableDocument(doc)
  const prefix = canonicalText(anchor.prefix), suffix = canonicalText(anchor.suffix)
  let best = null
  for (let offset = searchable.normalized.indexOf(exact); offset >= 0; offset = searchable.normalized.indexOf(exact, offset + 1)) {
    let score = 100
    if (prefix && searchable.normalized.slice(Math.max(0, offset - prefix.length), offset) === prefix) score += 35
    if (suffix && searchable.normalized.slice(offset + exact.length, offset + exact.length + suffix.length) === suffix) score += 35
    if (!best || score > best.score) best = { score, offset }
  }
  return best ? best.offset / Math.max(1, searchable.normalized.length) : null
}

// The first readable text at the top-left of the page that is showing.
export function anchorFromFrame(frameWindow, spineIndex, progress) {
  const doc = frameWindow?.document
  if (!doc?.body) return null
  const width = frameWindow.innerWidth || 0, height = frameWindow.innerHeight || 0
  for (const [fx, fy] of [[0.1, 0.08], [0.1, 0.2], [0.3, 0.08], [0.1, 0.4], [0.5, 0.3]]) {
    let node = null, offset = 0
    if (doc.caretRangeFromPoint) {
      const range = doc.caretRangeFromPoint(width * fx, height * fy)
      node = range?.startContainer; offset = range?.startOffset ?? 0
    } else if (doc.caretPositionFromPoint) {
      const caret = doc.caretPositionFromPoint(width * fx, height * fy)
      node = caret?.offsetNode; offset = caret?.offset ?? 0
    }
    if (node?.nodeType === 3 && node.data.trim()) {
      const anchor = anchorAroundTextNode(node, offset, spineIndex, progress)
      if (anchor) return anchor
    }
  }
  return null
}
