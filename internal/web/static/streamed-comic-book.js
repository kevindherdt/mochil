const MAX_COMIC_PAGES = 20_000

export function makeStreamedComicBook({ title, pageCount, pageURL }) {
  const count = Math.trunc(Number(pageCount))
  if (!Number.isFinite(count) || count < 1 || count > MAX_COMIC_PAGES)
    throw new Error('This comic has an invalid page count')
  if (typeof pageURL !== 'string' || !pageURL.includes('{pageNumber}'))
    throw new Error('This comic does not provide page streaming')

  const pages = new Map()
  const load = index => {
    if (pages.has(index)) return pages.get(index)
    const source = new URL(pageURL.replace('{pageNumber}', String(index)), location.origin)
    if (source.origin !== location.origin)
      throw new Error('Comic pages must come from this Mochil server')
    const safeSource = source.href.replaceAll('&', '&amp;').replaceAll('"', '&quot;')
    const documentURL = URL.createObjectURL(new Blob([
      `<!doctype html><html><head><meta charset="utf-8"></head><body><img src="${safeSource}" alt="Page ${index + 1}"></body><style>body{margin:0;background:#000}img{display:block}</style></html>`,
    ], { type: 'text/html' }))
    pages.set(index, documentURL)
    return documentURL
  }
  const unload = index => {
    const documentURL = pages.get(index)
    if (documentURL) URL.revokeObjectURL(documentURL)
    pages.delete(index)
  }
  const sections = Array.from({ length: count }, (_, index) => ({
    id: `page:${index + 1}`,
    load: () => load(index),
    unload: () => unload(index),
    size: 1,
  }))
  return {
    metadata: { title },
    sections,
    toc: [],
    rendition: { layout: 'pre-paginated' },
    resolveHref: href => ({ index: Math.max(0, sections.findIndex(section => section.id === href)) }),
    splitTOCHref: href => [href, null],
    getTOCFragment: document => document.documentElement,
    destroy: () => [...pages.keys()].forEach(unload),
  }
}
