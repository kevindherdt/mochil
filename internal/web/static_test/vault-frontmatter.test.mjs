import test from 'node:test'
import assert from 'node:assert/strict'
import {splitVaultFrontmatter} from '../static/vault-frontmatter.js'
import {createVaultMarkdownRenderer} from '../static/vault-markdown.js'

test('properties preserve original YAML and line endings without executing markup', () => {
  for(const newline of ['\n','\r\n','\r']) {
    const properties=['title: <script>alert(1)</script>','nested:','  tags: [one, two]','---not-a-delimiter',''].join(newline)
    const source=`\uFEFF---${newline}${properties}---${newline}# Body`
    assert.deepEqual(splitVaultFrontmatter(source),{properties,body:'# Body'})
    const html=createVaultMarkdownRenderer([],'vault')(source,'Note.md')
    assert.match(html,/<details class="vault-properties"><summary>Properties<\/summary>/)
    assert.match(html,/&lt;script&gt;/)
    assert.doesNotMatch(html,/<script>|<details[^>]* open/)
    assert.match(html,/<h1>Body<\/h1>/)
  }
})

test('unfinished and non-leading delimiters remain ordinary content', () => {
  for(const source of ['---\ntitle: Note','Text\n---\ntitle: Note\n---',' ---\ntitle: Note\n---','---\ntitle: Note\n---suffix'])
    assert.deepEqual(splitVaultFrontmatter(source),{properties:null,body:source})
})

test('empty properties do not create empty controls and closing delimiter can end file', () => {
  const render=createVaultMarkdownRenderer([],'vault')
  assert.equal(render('---\n---\n# Body','Note.md'),'<h1>Body</h1>')
  assert.deepEqual(splitVaultFrontmatter('---\nx: yes\n---'),{properties:'x: yes\n',body:''})
})
