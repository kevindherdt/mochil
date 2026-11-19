import test from 'node:test'
import assert from 'node:assert/strict'
import {createVaultPageResolver} from '../static/vault-page-resolver.js'

const page = (path, extra = {}) => ({id:path, path, library_id:'vault', format:'md', ...extra})
const entries = [page('Note.md'), page('A/Note.md'), page('B/Note.md'), page('B/Unique.markdown')]

test('short filename links prefer exact case independently of display titles', () => {
  const upper = page('Projects/Index.md', {title:'Future plans'})
  const lower = page('Other/index.md', {title:'Other note'})
  for (const values of [[upper,lower],[lower,upper]]) {
    const resolve=createVaultPageResolver(values,'vault')
    assert.equal(resolve('Journal/Today.md','Index'),upper)
    assert.equal(resolve('Journal/Today.md','index'),lower)
    assert.equal(resolve('Journal/Today.md','INDEX'),null)
  }
  const duplicate=page('Third/Index.md',{title:'Index'})
  const resolve=createVaultPageResolver([upper,duplicate],'vault')
  assert.equal(resolve('Journal/Today.md','Index'),null)
  assert.equal(resolve('Projects/Today.md','Index'),upper)
})

test('local page wins over root regardless of index order', () => {
  for (const list of [entries, [...entries].reverse()]) {
    const resolve = createVaultPageResolver(list, 'vault')
    assert.equal(resolve('A/Start.md', 'Note').path, 'A/Note.md')
    assert.equal(resolve('A/Start.md', '../Note.md').path, 'Note.md')
    assert.equal(resolve('A/Start.md', 'B/Unique').path, 'B/Unique.markdown')
    assert.equal(resolve('A/Start.md', 'Unique').path, 'B/Unique.markdown')
    assert.equal(resolve('A/Start.md', './Unique'), null)
  }
})

test('ambiguous basenames and case folded paths do not pick first entry', () => {
  const resolve = createVaultPageResolver([page('A/Note.md'), page('B/Note.md'), page('Case.md'), page('case.md')], 'vault')
  assert.equal(resolve('Start.md', 'Note'), null)
  assert.equal(resolve('Start.md', 'CASE'), null)
  assert.equal(resolve('Start.md', 'Case').path, 'Case.md')
  assert.equal(resolve('Start.md', 'case').path, 'case.md')
})

test('only vault pages can be resolved and paths cannot escape the vault', () => {
  const resolve = createVaultPageResolver([
    ...entries, page('Foreign.md', {library_id:'other'}),
    page('Image.png', {format:'png'}), page('Folder', {kind:'folder'}),
  ], 'vault')
  for (const target of ['../../Note','/Note','https://example.test/Note','Foreign','Image.png','Folder','bad\u0000Note'])
    assert.equal(resolve('A/Start.md', target), null, target)
})

test('encoded literal filenames are not decoded a second time', () => {
  const resolve = createVaultPageResolver([page('100%25.md'),page('日本語.md')], 'vault')
  assert.equal(resolve('Start.md', '100%25').path, '100%25.md')
  assert.equal(resolve('Start.md', '日本語').path, '日本語.md')
})
