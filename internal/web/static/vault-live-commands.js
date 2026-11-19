// Formatting commands for the wiki editor, as plain transaction specs over an
// EditorState so they work from buttons and from keys, and can be tested alone.

/** Wraps the selection in `before`/`after`, or removes them when already there. */
export function toggleWrap(state, before, after, placeholder = '') {
  const { from, to } = state.selection.main
  const outerFrom = from - before.length, outerTo = to + after.length
  if (outerFrom >= 0 && outerTo <= state.doc.length && state.sliceDoc(outerFrom, from) === before && state.sliceDoc(to, outerTo) === after) {
    return { changes: [{ from: outerFrom, to: from }, { from: to, to: outerTo }], selection: { anchor: outerFrom, head: to - before.length } }
  }
  const selected = state.sliceDoc(from, to) || placeholder
  const start = from + before.length
  return { changes: { from, to, insert: `${before}${selected}${after}` }, selection: { anchor: start, head: start + selected.length } }
}

/** Adds `prefix` to every selected line, or removes it when all of them have it. */
export function toggleLinePrefix(state, prefix) {
  const { from, to } = state.selection.main
  const first = state.doc.lineAt(from).number, last = state.doc.lineAt(to).number
  const lines = []
  for (let number = first; number <= last; number++) lines.push(state.doc.line(number))
  const all = lines.every(line => line.text.startsWith(prefix))
  const changes = lines.map(line => all ? { from: line.from, to: line.from + prefix.length } : { from: line.from, insert: prefix })
  return { changes }
}

/** `[selected](url)` with the address selected, ready to type. */
export function insertLink(state) {
  const { from, to } = state.selection.main
  const label = state.sliceDoc(from, to) || 'link'
  const urlStart = from + label.length + 3
  return { changes: { from, to, insert: `[${label}](url)` }, selection: { anchor: urlStart, head: urlStart + 3 } }
}

/** Makes every selected line a heading of `level`, or plain text when it already is one. */
export function setHeading(state, level) {
  const { from, to } = state.selection.main
  const prefix = `${'#'.repeat(level)} `
  const lines = []
  for (let number = state.doc.lineAt(from).number; number <= state.doc.lineAt(to).number; number++) lines.push(state.doc.line(number))
  const all = lines.every(line => line.text.startsWith(prefix))
  return { changes: lines.map(line => {
    const old = /^#{1,6}\s/.exec(line.text)?.[0].length || 0
    return { from: line.from, to: line.from + old, insert: all ? '' : prefix }
  }) }
}

/** 1. 2. 3. on the selected lines, or removed when they are all numbered already. */
export function toggleNumbered(state) {
  const { from, to } = state.selection.main
  const lines = []
  for (let number = state.doc.lineAt(from).number; number <= state.doc.lineAt(to).number; number++) lines.push(state.doc.line(number))
  const all = lines.every(line => /^\d+\.\s/.test(line.text))
  return { changes: lines.map((line, index) => all
    ? { from: line.from, to: line.from + /^\d+\.\s/.exec(line.text)[0].length }
    : { from: line.from, insert: `${index + 1}. ` }) }
}

const glyph = body => `<svg viewBox="0 0 24 24" aria-hidden="true">${body}</svg>`
const dots = (...ys) => ys.map(y => `<circle cx="4.5" cy="${y}" r="1" fill="currentColor" stroke="none"/>`).join('')

// `group` puts a divider between sets of tools; `text` tools show letters instead of an icon.
export const formatTools = [
  { id: 'bold', group: 'text', label: 'Bold', key: 'Mod-b', icon: glyph('<path d="M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z"/>'), run: state => toggleWrap(state, '**', '**', 'bold') },
  { id: 'italic', group: 'text', label: 'Italic', key: 'Mod-i', icon: glyph('<path d="M10 5h6M8 19h6M14 5l-4 14"/>'), run: state => toggleWrap(state, '_', '_', 'italic') },
  { id: 'strike', group: 'text', label: 'Strikethrough', icon: glyph('<path d="M5 12h14M16 7c-.9-1.2-2.2-1.7-4-1.7-2.1 0-3.7 1-3.7 2.7 0 3.8 7.6 1.8 7.6 6 0 1.8-1.7 2.7-3.9 2.7-1.8 0-3.3-.7-4.1-2.1"/>'), run: state => toggleWrap(state, '~~', '~~', 'text') },
  { id: 'code', group: 'text', label: 'Code', key: 'Mod-e', icon: glyph('<path d="m8 8-4 4 4 4M16 8l4 4-4 4"/>'), run: state => toggleWrap(state, '`', '`', 'code') },
  { id: 'h1', group: 'heading', label: 'Heading 1', text: 'H1', run: state => setHeading(state, 1) },
  { id: 'h2', group: 'heading', label: 'Heading 2', text: 'H2', run: state => setHeading(state, 2) },
  { id: 'h3', group: 'heading', label: 'Heading 3', text: 'H3', run: state => setHeading(state, 3) },
  { id: 'link', group: 'insert', label: 'Link', key: 'Mod-k', icon: glyph('<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>'), run: insertLink },
  { id: 'wiki', group: 'insert', label: 'Link to page', key: 'Mod-Shift-k', icon: glyph('<path d="M9 4H6v16h3M15 4h3v16h-3"/>'), run: state => toggleWrap(state, '[[', ']]', 'Page') },
  { id: 'image', group: 'insert', label: 'Add image', icon: glyph('<rect x="4" y="5" width="16" height="14" rx="2"/><circle cx="9" cy="10" r="1.5"/><path d="m20 16-4.5-4.5L8 19"/>'), action: 'image' },
  { id: 'template', group: 'insert', label: 'Insert template', action: 'template', icon: glyph('<path d="M7 3h8l4 4v14H7zM14 3v5h5M10 13h6M10 17h4"/>') },
  { id: 'list', group: 'block', label: 'Bulleted list', icon: glyph(`<path d="M9 7h11M9 12h11M9 17h11"/>${dots(7, 12, 17)}`), run: state => toggleLinePrefix(state, '- ') },
  { id: 'numbered', group: 'block', label: 'Numbered list', icon: glyph('<path d="M10 7h10M10 12h10M10 17h10M4 6l1.5-1v4M4 15h3l-3 4h3"/>'), run: toggleNumbered },
  { id: 'task', group: 'block', label: 'Task list', icon: glyph('<rect x="4" y="5" width="6" height="6" rx="1.5"/><path d="m5.5 8 1.2 1.2L8.8 7M13 8h7M4 16h6m3 0h7"/>'), run: state => toggleLinePrefix(state, '- [ ] ') },
  { id: 'quote', group: 'block', label: 'Quote', icon: glyph('<path d="M9 8c-2 0-3 1.3-3 3v5h4v-4H7.5M18 8c-2 0-3 1.3-3 3v5h4v-4h-2.5"/>'), run: state => toggleLinePrefix(state, '> ') },
]

/** The toolbar's markup: one button per tool, a divider between groups. */
export function toolbarHTML(tools = formatTools) {
  let previous = ''
  return tools.map(tool => {
    const divider = previous && previous !== tool.group ? '<i class="vault-format-divider" aria-hidden="true"></i>' : ''
    previous = tool.group
    const hint = `${tool.label}${tool.key ? ` (${tool.key.replace('Mod', 'Ctrl').replace('Shift-', 'Shift+').replace(/-/g, '+')})` : ''}`
    return `${divider}<button type="button" data-format="${tool.id}" title="${hint}" aria-label="${tool.label}">${tool.icon || `<b>${tool.text}</b>`}</button>`
  }).join('')
}
