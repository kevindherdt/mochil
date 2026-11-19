import test from 'node:test'
import assert from 'node:assert/strict'
import {parseVaultReference, jumpToVaultHeading} from '../static/vault-heading.js'
import {createVaultMarkdownRenderer} from '../static/vault-markdown.js'

test('wiki references preserve page and heading with one decoding step', () => {
  assert.deepEqual(parseVaultReference('#vault:Notes%2FPage%23A%20heading'), {page:'Notes/Page',heading:'A heading'})
  assert.deepEqual(parseVaultReference('#vault:%23日本語'), {page:'',heading:'日本語'})
  assert.deepEqual(parseVaultReference('#vault:100%2525%23Heading'), {page:'100%25',heading:'Heading'})
  assert.equal(parseVaultReference('#vault:bad%ZZ'), null)
  assert.equal(parseVaultReference('https://example.test'), null)
  const render = createVaultMarkdownRenderer([], 'vault')
  assert.match(render('[[Page#Heading|Go]] [[#Local]]', 'Page.md'), /#vault:Page%23Heading/)
  assert.match(render('[[#Local]]', 'Page.md'), /#vault:%23Local/)
})

test('heading jump matches rendered Unicode text and focuses without a second scroll', () => {
  const calls = []
  const heading = {textContent:'Café  日本語', setAttribute:(...args)=>calls.push(args),
    scrollIntoView:options=>calls.push(options), focus:options=>calls.push(options)}
  const pane = {querySelectorAll:()=>[heading]}
  assert.equal(jumpToVaultHeading(pane, ' cafe\u0301 日本語 '), true)
  assert.deepEqual(calls, [['tabindex','-1'], {block:'start',behavior:'instant'}, {preventScroll:true}])
  assert.equal(jumpToVaultHeading(pane, 'Missing'), false)
  assert.equal(jumpToVaultHeading(pane, ''), false)
  assert.equal(calls.length, 3)
})
