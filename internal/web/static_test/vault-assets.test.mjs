import test from 'node:test'
import assert from 'node:assert/strict'
import {createVaultAssetResolver} from '../static/vault-assets.js'

const entry = (id,path,extra={}) => ({id,path,library_id:'vault',format:'png',...extra})
const resolve = createVaultAssetResolver([
  entry('other','Other/cover.png'), entry('local','Notes/Assets/cover.png'),
  entry('remote','Other/remote.png',{library_id:'another'}),
  entry('svg','vector.svg',{format:'svg'}),
], 'vault')

test('vault image references preserve relative root and shorthand matching', () => {
  assert.equal(resolve('Notes/Page.md','Assets/cover.png').entry.id,'local')
  assert.equal(resolve('Elsewhere/Page.md','Notes/Assets/cover.png').entry.id,'local')
  assert.equal(resolve('Notes/Page.md','../Other/cover.png').entry.id,'other')
  assert.equal(resolve('Page.md','cover.png'),null)
})

test('explicit missing paths cannot substitute unrelated attachments', () => {
  for (const target of ['Missing/cover.png','./cover.png','Missing\\cover.png|300','../../cover.png']) {
    assert.equal(resolve('Page.md',target),null,target)
  }
})

test('remote schemes absolute paths cross-vault entries and active formats are rejected', () => {
  for (const target of ['https://host/cover.png','javascript:cover.png','data:image/png,x','//host/cover.png','/cover.png','remote.png','vector.svg']) {
    assert.equal(resolve('Page.md',target),null,target)
  }
})

test('asset URL is constructed and escaped rather than trusted from metadata', () => {
  const lookup=createVaultAssetResolver([entry('image','A & B.png',{open_url:'https://tracker.test'})],'vault')
  assert.equal(lookup('Page.md','A & B.png').url,'/documents/vault/image/A%20%26%20B.png')
})

test('relative images win over root images independently of index order', () => {
  const images=[entry('root','cover.png'),entry('local','Notes/cover.png')]
  for (const list of [images,[...images].reverse()]) {
    const lookup=createVaultAssetResolver(list,'vault')
    assert.equal(lookup('Notes/Page.md','cover.png').entry.id,'local')
    assert.equal(lookup('Notes/Page.md','./cover.png').entry.id,'local')
    assert.equal(lookup('Notes/Page.md','../cover.png').entry.id,'root')
    assert.equal(lookup('Missing/Page.md','./cover.png'),null)
    assert.equal(lookup('Notes/Page.md','../../cover.png'),null)
  }
})

test('duplicate shorthand and case folded paths are never selected by index order', () => {
  const lookup=createVaultAssetResolver([
    entry('a','A/cover.png'),entry('b','B/cover.png'),
    entry('upper','Notes/COVER.png'),entry('lower','Notes/cover.png'),
  ],'vault')
  assert.equal(lookup('Page.md','cover.png'),null)
  assert.equal(lookup('Notes/Page.md','cover.png').entry.id,'lower')
  assert.equal(lookup('Notes/Page.md','COVER.png').entry.id,'upper')
  assert.equal(lookup('Notes/Page.md','Cover.png'),null)
  assert.equal(lookup('Notes/Page.md','bad\u007f.png'),null)
})

test('conflicting exact paths are refused while repeated identity remains resolvable', () => {
  const image=entry('one','Notes/cover.png')
  assert.equal(createVaultAssetResolver([image,image],'vault')('Notes/Page.md','cover.png').entry.id,'one')
  const conflict=entry('two',image.path)
  for (const list of [[image,conflict],[conflict,image]]) {
    assert.equal(createVaultAssetResolver(list,'vault')('Notes/Page.md','cover.png'),null)
  }
})
