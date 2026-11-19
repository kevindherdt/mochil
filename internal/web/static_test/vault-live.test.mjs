import assert from 'node:assert/strict'
import test from 'node:test'
import { EditorState, markdown, markdownLanguage, syntaxTree } from '../static/vendor/codemirror/codemirror.js'
import { previewSpecs, frontmatterEnd } from '../static/vault-live-preview.js'
import { insertLink, toggleLinePrefix, toggleWrap, setHeading, toggleNumbered, toolbarHTML } from '../static/vault-live-commands.js'

const stateOf = (doc, anchor = 0, head = anchor) =>
  EditorState.create({ doc, selection: { anchor, head }, extensions: [markdown({ base: markdownLanguage })] })
const specsOf = (doc, cursor = doc.length) => {
  const state = stateOf(doc, cursor)
  return previewSpecs({ tree: syntaxTree(state), doc: state.doc, ranges: state.selection.ranges.map(r => ({ from: r.from, to: r.to })) })
}
const hidden = (doc, specs) => specs.filter(spec => spec.kind === 'hide').sort((a, b) => a.from - b.from).map(spec => doc.slice(spec.from, spec.to))

test('heading marks are hidden until the cursor is on that line', () => {
  const doc = '# Title\n\nBody'
  assert.deepEqual(hidden(doc, specsOf(doc, doc.length)), ['# '])
  assert.deepEqual(hidden(doc, specsOf(doc, 3)), [])
  assert.ok(specsOf(doc).some(spec => spec.kind === 'line' && spec.className.includes('cm-md-h1')))
})

test('bold and italic show their words and hide the stars until touched', () => {
  const doc = 'a **bold** and _it_ end'
  assert.deepEqual(hidden(doc, specsOf(doc, doc.length)), ['**', '**', '_', '_'])
  assert.deepEqual(hidden(doc, specsOf(doc, 6)), ['_', '_'], 'only the touched run shows its marks')
  const classes = specsOf(doc).filter(spec => spec.kind === 'mark').map(spec => spec.className)
  assert.ok(classes.includes('cm-md-strong') && classes.includes('cm-md-em'))
})

test('links read as their words, addresses stay in the file', () => {
  const doc = 'see [docs](https://x.y/z) now'
  assert.deepEqual(hidden(doc, specsOf(doc, doc.length)), ['[', '](https://x.y/z)'])
  assert.deepEqual(hidden(doc, specsOf(doc, 8)), [])
})

test('wiki links hide their brackets and the target when an alias is given', () => {
  const doc = 'go to [[Plan]] or [[Plan|the plan]] later'
  const specs = specsOf(doc, doc.length)
  assert.deepEqual(hidden(doc, specs), ['[[', ']]', '[[', 'Plan|', ']]'])
  assert.equal(specs.filter(spec => spec.className === 'cm-md-wiki').length, 2)
})

test('code is left alone: no tags or wiki links inside it', () => {
  const doc = '`#notatag [[x]]` and #real\n\n```\n#nope\n```\n'
  const tags = specsOf(doc, 0).filter(spec => spec.className === 'cm-md-tag').map(spec => doc.slice(spec.from, spec.to))
  assert.deepEqual(tags, ['#real'])
})

test('a heading is not a tag, and frontmatter is recognised', () => {
  assert.equal(specsOf('# Heading\n').some(spec => spec.className === 'cm-md-tag'), false)
  const doc = '---\ntags: [a]\n---\nBody'
  assert.equal(frontmatterEnd(stateOf(doc).doc), 17)
  assert.equal(frontmatterEnd(stateOf('no front\n---\n').doc), 0)
})

test('quotes, tasks and rules get their classes', () => {
  const doc = '> quoted\n\n- [x] done\n- [ ] todo\n\n---\n'
  const classes = specsOf(doc, 0).map(spec => spec.className).filter(Boolean)
  for (const wanted of ['cm-md-quote', 'cm-md-task-done', 'cm-md-done', 'cm-md-hr']) assert.ok(classes.some(name => name.includes(wanted)), wanted)
})

test('wrap toggles: add, remove, and placeholder', () => {
  const apply = (state, spec) => state.update(spec).state
  let state = stateOf('word', 0, 4)
  state = apply(state, toggleWrap(state, '**', '**', 'bold'))
  assert.equal(state.doc.toString(), '**word**')
  assert.equal(state.sliceDoc(state.selection.main.from, state.selection.main.to), 'word')
  state = apply(state, toggleWrap(state, '**', '**', 'bold'))
  assert.equal(state.doc.toString(), 'word')
  const empty = stateOf('')
  assert.equal(apply(empty, toggleWrap(empty, '_', '_', 'italic')).doc.toString(), '_italic_')
})

test('line prefixes toggle across the selection and links select their address', () => {
  const apply = (state, spec) => state.update(spec).state
  let state = stateOf('one\ntwo', 0, 7)
  state = apply(state, toggleLinePrefix(state, '- '))
  assert.equal(state.doc.toString(), '- one\n- two')
  state = stateOf(state.doc.toString(), 0, 11)
  assert.equal(apply(state, toggleLinePrefix(state, '- ')).doc.toString(), 'one\ntwo')
  const link = stateOf('page', 0, 4)
  const next = apply(link, insertLink(link))
  assert.equal(next.doc.toString(), '[page](url)')
  assert.equal(next.sliceDoc(next.selection.main.from, next.selection.main.to), 'url')
})

test('blank lines between blocks are slimmed, except where the cursor is and inside code', () => {
  const doc = 'one\n\ntwo\n\n```\n\ncode\n```\n'
  const blanks = specsOf(doc, 0).filter(spec => spec.className === 'cm-md-blank').map(spec => spec.from)
  assert.deepEqual(blanks, [4, 9, 24], 'the blank inside the code fence at 14 is kept')
  assert.deepEqual(specsOf(doc, 4).filter(spec => spec.className === 'cm-md-blank').map(spec => spec.from), [9, 24])
})

const apply = (state, spec) => state.update(spec).state.doc.toString()

test('heading buttons replace the old level instead of stacking', () => {
  assert.equal(apply(stateOf('## Title'), setHeading(stateOf('## Title'), 1)), '# Title')
  assert.equal(apply(stateOf('Title'), setHeading(stateOf('Title'), 3)), '### Title')
  assert.equal(apply(stateOf('# Title'), setHeading(stateOf('# Title'), 1)), 'Title')
})

test('numbered lists count up and toggle off', () => {
  const doc = 'a\nb\nc'
  const on = apply(stateOf(doc, 0, doc.length), toggleNumbered(stateOf(doc, 0, doc.length)))
  assert.equal(on, '1. a\n2. b\n3. c')
  assert.equal(apply(stateOf(on, 0, on.length), toggleNumbered(stateOf(on, 0, on.length))), doc)
})

test('the toolbar is icon buttons in labelled groups', () => {
  const html = toolbarHTML()
  assert.match(html, /aria-label="Bold"/)
  assert.match(html, /title="Bold \(Ctrl\+b\)"/)
  assert.equal((html.match(/vault-format-divider/g) || []).length, 3)
  assert.ok(!html.includes('>Bold<'), 'no text pills')
})

test('![[picture]] becomes an image only when it resolves and the cursor is elsewhere', () => {
  const doc = 'Hello\n![[shot.png|300]]\n'
  const state = stateOf(doc, 0)
  const run = (cursor, isImage) => {
    const at = stateOf(doc, cursor)
    return previewSpecs({ tree: syntaxTree(at), doc: at.doc, ranges: [{ from: cursor, to: cursor }], isImage })
  }
  const image = run(0, target => target === 'shot.png').find(spec => spec.kind === 'image')
  assert.deepEqual([image.from, image.to, image.target, image.width], [6, 23, 'shot.png', 300])
  assert.ok(!run(10, () => true).some(spec => spec.kind === 'image'), 'cursor inside shows the text')
  assert.ok(!run(0, () => false).some(spec => spec.kind === 'image'), 'unknown picture stays text')
  assert.ok(state)
})
