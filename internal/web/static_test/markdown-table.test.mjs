import test from 'node:test'
import assert from 'node:assert/strict'
import {markdownTableCells, markdownTableHeader} from '../static/markdown-table.js'
import {renderMarkdown} from '../static/markdown-renderer.js'
import {createVaultMarkdownRenderer} from '../static/vault-markdown.js'

test('pipe tables handle optional boundaries escaped pipes and alignment', () => {
  assert.deepEqual(markdownTableCells('| A | B |'), ['A','B'])
  assert.deepEqual(markdownTableCells('A \\| B | C'), ['A | B','C'])
  assert.deepEqual(markdownTableHeader('A | B | C', ':--- | :---: | ---:').alignments, ['left','center','right'])
  assert.equal(markdownTableHeader('A | B', '--- | nope'), null)
  assert.equal(markdownTableCells(Array(66).fill('x').join('|')), null)
})

test('wiki tables preserve inline links and escaping without affecting other renderers', () => {
  const source='| Name | Note |\n| --- | ---: |\n| **One** | [[Page]] |\n| <script> | `code` |'
  const html=createVaultMarkdownRenderer([],'vault')(source,'Note.md')
  assert.match(html,/<table>/)
  assert.match(html,/scope="col"/)
  assert.match(html,/<strong>One<\/strong>/)
  assert.match(html,/#vault:Page/)
  assert.match(html,/&lt;script&gt;/)
  assert.doesNotMatch(renderMarkdown(source),/<table>/)
  assert.doesNotMatch(createVaultMarkdownRenderer([],'vault')('```\n'+source+'\n```','Note.md'),/<table>/)
})

test('mismatched rows remain visible outside the table', () => {
  const html=renderMarkdown('A | B\n--- | ---\nx | y\nExtra | column | retained',{tables:true})
  assert.match(html,/<td[^>]*>x<\/td>/)
  assert.match(html,/<p>Extra \| column \| retained<\/p>/)
})

test('odd backslash runs escape pipes while even runs preserve column boundaries', () => {
  for (let count = 0; count <= 12; count++) {
    const slashes = '\\'.repeat(count)
    const source = `A${slashes}| B | C`
    assert.deepEqual(markdownTableCells(source), count % 2
      ? [`A${'\\'.repeat(count - 1)}| B`, 'C']
      : [`A${slashes}`, 'B', 'C'])
    const ending = `| A | B${slashes}|`
    assert.deepEqual(markdownTableCells(ending), count % 2
      ? ['A', `B${'\\'.repeat(count - 1)}|`]
      : ['A', `B${slashes}`])
  }
})

test('escaped pipes inside inline code remain in one table cell', () => {
  const html=renderMarkdown('A | B\n--- | ---\n`left\\|right` | next',{tables:true})
  assert.match(html,/<code>left\|right<\/code>/)
  assert.equal((html.match(/<td /g)||[]).length,2)
})
