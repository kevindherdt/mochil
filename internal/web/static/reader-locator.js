import { anchorAroundTextNode, canonicalText, searchableDocument } from './text-anchor.js?v=1'
export function createReaderLocator(view) {
function webAnchorFromLocation(detail) {
  const range = detail?.range
  if (!range) return null
  const content = (view.renderer?.getContents?.() || []).find(item => item.doc === range.startContainer?.ownerDocument)
  const spineIndex = Number.isInteger(detail?.index) ? detail.index : (Number.isInteger(content?.index) ? content.index : 0)
  let node = range.startContainer
  let offset = range.startOffset
  if (node.nodeType !== Node.TEXT_NODE) {
    const walker = node.ownerDocument.createTreeWalker(node, NodeFilter.SHOW_TEXT)
    node = walker.nextNode()
    offset = 0
  }
  if (!node) return null
  return anchorAroundTextNode(node, offset, spineIndex, Number.isFinite(detail?.fraction) ? detail.fraction : (view.lastLocation?.fraction || 0))
}

async function resolveTextAnchor(anchor) {
  const exact = canonicalText(anchor?.exact || anchor?.text)
  const sections = view.book?.sections || []
  if (exact.length < 24 || !sections.length) return null
  const preferred = Number.isInteger(anchor.spine_index) ? anchor.spine_index : 0
  const nearby = [preferred, preferred - 1, preferred + 1].filter((value, index, all) => value >= 0 && value < sections.length && all.indexOf(value) === index)
  const remaining = sections.map((_, index) => index).filter(index => !nearby.includes(index))
  const prefix = canonicalText(anchor.prefix)
  const suffix = canonicalText(anchor.suffix)
  let best = null
  for (const index of [...nearby, ...remaining]) {
    try {
      const doc = await sections[index].createDocument()
      const searchable = searchableDocument(doc)
      let offset = searchable.normalized.indexOf(exact)
      while (offset >= 0) {
        const point = searchable.points[offset]
        if (point) {
          const before = searchable.normalized.slice(Math.max(0, offset - prefix.length - 16), offset)
          const after = searchable.normalized.slice(offset + exact.length, offset + exact.length + suffix.length + 16)
          const approximate = (index + offset / Math.max(1, searchable.normalized.length)) / sections.length
          let score = 100 + (index === preferred ? 50 : 0) - Math.abs(approximate - Number(anchor.progress || 0)) * 20
          if (prefix && before.endsWith(prefix)) score += 35
          if (suffix && after.startsWith(suffix)) score += 35
          if (!best || score > best.score) {
            const range = doc.createRange()
            range.setStart(point.node, Math.min(point.offset, point.node.data.length))
            range.collapse(true)
            best = { score, cfi: view.getCFI(index, range), accuracy: 'text' }
          }
        }
        offset = searchable.normalized.indexOf(exact, offset + 1)
      }
      if (best?.score >= 215) break
    } catch (error) {
      console.warn('Could not search EPUB section for synced text', error)
    }
  }
  return best
}

// KOReader stores EPUB locations as CREngine XPointers. EPUB DocFragment
// numbers follow spine order, while Foliate uses a zero-based spine index and
// EPUB CFI. Resolve the structural path in the source document, then let
// Foliate convert the resulting DOM range to its native CFI.
function parseKoReaderXPointer(value) {
  if (typeof value !== 'string') return null
  const match = value.match(/^\/body\/DocFragment\[(\d+)](?:\.\d+)?(?:\/body(?:\/(.*))?)?$/)
  if (!match) return null
  const fragment = Number(match[1])
  if (!Number.isSafeInteger(fragment) || fragment < 1) return null
  const steps = (match[2] || '').split('/').filter(Boolean).map(part => {
    const text = part.match(/^text\(\)(?:\[(\d+)])?(?:\.(\d+))?$/)
    if (text) return { text: true, index: Number(text[1] || 1), offset: Number(text[2] || 0) }
    const element = part.match(/^([^\[.]+)(?:\[(\d+)])?$/)
    return element ? { name: element[1].toLowerCase(), index: Number(element[2] || 1) } : null
  })
  return steps.some(step => !step) ? null : { index: fragment - 1, steps }
}

function resolveXPointerNode(doc, steps) {
  let node = doc.body
  for (const step of steps) {
    if (step.text) {
      const texts = [...node.childNodes].filter(child => child.nodeType === Node.TEXT_NODE)
      node = texts[step.index - 1]
      if (!node) return null
      return { node, offset: Math.min(step.offset, node.data.length) }
    }
    const children = [...node.children].filter(child => child.localName?.toLowerCase() === step.name)
    node = children[step.index - 1]
    if (!node) return null
  }
  return { node, offset: 0 }
}

async function resolveKoReaderPosition(progress) {
  const parsed = parseKoReaderXPointer(progress)
  const section = parsed && view.book?.sections?.[parsed.index]
  if (!section) return null
  try {
    const doc = await section.createDocument()
    const point = resolveXPointerNode(doc, parsed.steps)
    if (point) {
      const range = doc.createRange()
      range.setStart(point.node, point.offset)
      range.collapse(true)
      return { cfi: view.getCFI(parsed.index, range), accuracy: 'exact' }
    }
    // CREngine may normalize wrapper elements differently from the browser.
    // The DocFragment still identifies the correct EPUB spine item.
    return { cfi: view.getCFI(parsed.index), accuracy: 'chapter' }
  } catch (error) {
    console.warn('Could not translate KOReader XPointer', error)
    return { cfi: view.getCFI(parsed.index), accuracy: 'chapter' }
  }
}

  return { webAnchorFromLocation, resolveTextAnchor, resolveKoReaderPosition }
}
