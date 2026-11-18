/** Small pipe-table parser; unsupported/oversized rows remain ordinary Markdown. */
export function markdownTableCells(line) {
  if (!line?.includes('|')) return null
  let text = line.trim()
  if (text.startsWith('|')) text = text.slice(1)
  // Do not mistake an escaped final pipe for the optional table boundary.
  if (text.endsWith('|') && !/(?:^|[^\\])(?:\\\\)*\\\|$/.test(text)) text = text.slice(0, -1)
  const cells = []
  let value = ''
  for (let index = 0; index < text.length; index++) {
    const character = text[index]
    if (character === '\\') {
      let end = index + 1
      while (text[end] === '\\') end++
      const count = end - index
      if (text[end] === '|' && count % 2 === 1) {
        value += '\\'.repeat(count - 1) + '|'
        index = end
      } else {
        value += '\\'.repeat(count)
        index = end - 1
      }
      continue
    }
    if (character === '|') {
      cells.push(value.trim()); value = ''
      if (cells.length >= 64) return null
    } else value += character
  }
  cells.push(value.trim())
  return cells
}

export function markdownTableHeader(line, divider) {
  const headers = markdownTableCells(line), markers = markdownTableCells(divider)
  if (!headers || !markers || headers.length !== markers.length || !markers.every(cell => /^:?-{3,}:?$/.test(cell))) return null
  return {headers, alignments: markers.map(cell => cell.startsWith(':') && cell.endsWith(':') ? 'center' : cell.endsWith(':') ? 'right' : 'left')}
}
