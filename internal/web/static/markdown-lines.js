/** Read normalized Markdown lines without allocating a second document and a full line array. */
export function createMarkdownLineCursor(source) {
  const text = String(source ?? '')
  const breaks = /\r\n?|\n/g
  let offset = 0, currentStart = 0, finished = false
  function next() {
    if (finished) return null
    const match = breaks.exec(text)
    if (!match) { finished = true; const line = text.slice(offset); offset = text.length; return line }
    const line = text.slice(offset, match.index)
    offset = breaks.lastIndex
    return line
  }
  let current = next()
  return {
    peek: () => current,
    position: () => currentStart,
    take: () => { const line = current; currentStart = offset; current = next(); return line },
  }
}
