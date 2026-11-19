// Decides how a Markdown note looks while it is being edited: headings, bold and
// links are styled in place, and the punctuation that makes them is hidden until
// the cursor reaches it. Pure over a syntax tree and a document, so it can be
// tested without a browser; vault-live-editor.js turns the result into decorations.

const quoteMark = /^>\s?/
const wikiLink = /\[\[([^\]\n|#]*)(#[^\]\n|]*)?(\|[^\]\n]*)?\]\]/g
const tag = /(^|[\s(])#([\p{L}_][\p{L}\p{N}_/-]*)/gu

/**
 * @param {{tree: {iterate: Function}, doc: object, ranges: {from:number,to:number}[], visible?: {from:number,to:number}}} input
 * @returns {{kind: 'hide'|'mark'|'line', from: number, to: number, className?: string}[]}
 */
export function previewSpecs({ tree, doc, ranges, visible = { from: 0, to: doc.length }, isImage = () => false }) {
  const specs = []
  const touches = (from, to) => ranges.some(range => range.from <= to && range.to >= from)
  const lineTouched = position => { const line = doc.lineAt(position); return touches(line.from, line.to) }
  const hide = (from, to) => { if (to > from) specs.push({ kind: 'hide', from, to }) }
  const mark = (from, to, className) => { if (to > from) specs.push({ kind: 'mark', from, to, className }) }
  const line = (position, className) => specs.push({ kind: 'line', from: doc.lineAt(position).from, to: doc.lineAt(position).from, className })
  const linesOf = (from, to, className) => {
    const first = Math.max(from, visible.from), last = Math.min(to, visible.to)
    for (let position = first; position <= last;) {
      const current = doc.lineAt(position)
      line(current.from, className)
      position = current.to + 1
    }
  }
  const afterSpace = position => (doc.sliceString(position, position + 1) === ' ' ? position + 1 : position)
  const codeRanges = []
  const wikiRanges = []

  // Wiki links and tags are not Markdown syntax, so they are found in the text.
  const text = doc.sliceString(visible.from, visible.to)
  tree.iterate({
    from: visible.from, to: visible.to,
    enter: node => {
      if (node.name === 'FencedCode' || node.name === 'InlineCode' || node.name === 'CodeBlock') codeRanges.push([node.from, node.to])
    },
  })
  const inCode = position => codeRanges.some(([from, to]) => position >= from && position < to)
  for (const match of text.matchAll(wikiLink)) {
    const from = visible.from + match.index, to = from + match[0].length
    if (inCode(from)) continue
    wikiRanges.push([from, to])
    const open = from + 2, close = to - 2
    const aliasStart = match[3] ? close - match[3].length : close
    const active = touches(from, to)
    // ![[picture.png]] shows the picture until the cursor comes near it.
    if (doc.sliceString(from - 1, from) === '!' && !active && isImage(match[1].trim())) {
      const width = /^\d{1,4}$/.test(match[3]?.slice(1) || '') ? Number(match[3].slice(1)) : 0
      specs.push({ kind: 'image', from: from - 1, to, target: match[1].trim(), width })
      continue
    }
    mark(open, close, 'cm-md-wiki')
    if (!active) {
      hide(from, open)
      hide(close, to)
      // [[Target|shown]] reads as "shown"; the target stays in the file.
      if (match[3]) hide(open, aliasStart + 1)
    }
  }
  for (const match of text.matchAll(tag)) {
    const from = visible.from + match.index + match[1].length
    const lineStart = doc.lineAt(from).from
    if (inCode(from) || wikiRanges.some(([a, b]) => from >= a && from < b) || /^#{1,6}\s/.test(doc.sliceString(lineStart, from + 2))) continue
    mark(from, from + 1 + match[2].length, 'cm-md-tag')
  }

  // A blank line between blocks is spacing, not text: keep it slim, as the page will.
  for (let position = visible.from; position <= visible.to;) {
    const current = doc.lineAt(position)
    if (current.length === 0 && !inCode(current.from) && !lineTouched(current.from)) line(current.from, 'cm-md-blank')
    position = current.to + 1
  }

  const front = frontmatterEnd(doc)
  if (front > 0) linesOf(0, front, 'cm-md-frontmatter')

  tree.iterate({
    from: visible.from, to: visible.to,
    enter: ({ name, from, to, node }) => {
      let match
      if ((match = /^ATXHeading([1-6])$/.exec(name))) line(from, `cm-md-h cm-md-h${match[1]}`)
      else if ((match = /^SetextHeading([12])$/.exec(name))) line(from, `cm-md-h cm-md-h${match[1]}`)
      else if (name === 'HeaderMark' && node.parent?.name.startsWith('ATX')) {
        if (!lineTouched(from)) hide(from, afterSpace(to))
      } else if (name === 'StrongEmphasis') mark(from, to, 'cm-md-strong')
      else if (name === 'Emphasis') mark(from, to, 'cm-md-em')
      else if (name === 'Strikethrough') mark(from, to, 'cm-md-strike')
      else if (name === 'InlineCode') mark(from, to, 'cm-md-code')
      else if (name === 'EmphasisMark' || name === 'StrikethroughMark' || (name === 'CodeMark' && node.parent?.name === 'InlineCode')) {
        if (!touches(node.parent.from, node.parent.to)) hide(from, to)
      } else if ((name === 'Link' || name === 'Image') && !wikiRanges.some(([a, b]) => from >= a - 1 && to <= b + 1)) {
        const marks = []
        for (let child = node.firstChild; child; child = child.nextSibling) if (child.name === 'LinkMark') marks.push(child)
        const textEnd = marks[1] ? marks[1].from : to
        mark(marks[0] ? marks[0].to : from, textEnd, name === 'Image' ? 'cm-md-image' : 'cm-md-link')
        if (!touches(from, to)) {
          // Show the words, keep the address in the file.
          hide(from, marks[0] ? marks[0].to : from)
          hide(textEnd, to)
        }
      } else if (name === 'Blockquote') linesOf(from, to, 'cm-md-quote')
      else if (name === 'QuoteMark') {
        if (!lineTouched(from)) hide(from, afterSpace(to))
      } else if (name === 'ListMark') mark(from, to, 'cm-md-listmark')
      else if (name === 'TaskMarker') {
        const done = /x/i.test(doc.sliceString(from, to))
        mark(from, to, done ? 'cm-md-task cm-md-task-done' : 'cm-md-task')
        if (done) line(from, 'cm-md-done')
      } else if (name === 'FencedCode') linesOf(from, to, 'cm-md-fence')
      else if (name === 'CodeMark' && node.parent?.name === 'FencedCode') line(from, 'cm-md-fence-mark')
      else if (name === 'HorizontalRule') {
        line(from, 'cm-md-hr')
        if (!lineTouched(from)) hide(from, to)
      }
    },
  })
  return specs
}

/** End of a leading `---` property block, or 0 when the note has none. */
export function frontmatterEnd(doc) {
  if (doc.length < 8 || doc.sliceString(0, 4) !== '---\n') return 0
  const lines = Math.min(doc.lines, 80)
  for (let number = 2; number <= lines; number++) {
    const line = doc.line(number)
    if (line.text.trim() === '---') return line.to
  }
  return 0
}
