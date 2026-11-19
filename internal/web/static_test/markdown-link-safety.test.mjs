import test from 'node:test'
import assert from 'node:assert/strict'
import {renderMarkdown} from '../static/markdown-renderer.js'

test('backslash authority cannot masquerade as a local Markdown link',()=>{
  const destination='/\\outside.example/path'
  assert.equal(new URL(destination,'https://library.example').hostname,'outside.example')
  assert.doesNotMatch(renderMarkdown(`[link](${destination})`),/href=/)
})

test('malformed URL backslashes and embedded controls never become anchors',()=>{
  for(const target of ['/\\outside.example','https://trusted.example\\@outside.example','/path\u0000name','/path\u007fname']){
    assert.doesNotMatch(renderMarkdown(`[link](${target})`),/href=/,target)
  }
})

test('normal local fragment and explicit external links remain available',()=>{
  for(const target of ['/books','./page','../page','#chapter','https://example.test/page','mailto:reader@example.test']){
    assert.match(renderMarkdown(`[link](${target})`),/href=/,target)
  }
})
