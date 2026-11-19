import test from 'node:test'
import assert from 'node:assert/strict'
import {changeVaultTask} from '../static/vault-task.js'
import {createVaultMarkdownRenderer} from '../static/vault-markdown.js'

test('rendered task offsets target only actual body markers with original line endings', () => {
  for(const newline of ['\n','\r\n','\r']) {
    const source=['\uFEFF---','example: - [ ] repeat','---','```','- [ ] repeat','```','- [ ] repeat','- [X] repeat'].join(newline)
    const render=createVaultMarkdownRenderer([],'vault')
    const html=render(source,'Page.md',{writable:true})
    const offsets=[...html.matchAll(/data-vault-task="(\d+)"/g)].map(match=>Number(match[1]))
    assert.equal(offsets.length,2)
    assert.equal(offsets[0],source.lastIndexOf('- [ ] repeat')+3)
    assert.equal(offsets[1],source.lastIndexOf('- [X] repeat')+3)
    assert.equal(changeVaultTask(source,offsets[0],true),source.slice(0,offsets[0])+'x'+source.slice(offsets[0]+1))
    assert.equal(changeVaultTask(source,offsets[1],false),source.slice(0,offsets[1])+' '+source.slice(offsets[1]+1))
    assert.doesNotMatch(html,/<input[^>]*disabled/)
    assert.match(render(source,'Page.md'),/<input[^>]*disabled/)
  }
})

test('task updates reject offsets that do not point at a valid marker', () => {
  for(const offset of [-1,0,1,2,4,5,999,NaN,3.5])
    assert.throws(()=>changeVaultTask('- [ ] text',offset,true))
  assert.throws(()=>changeVaultTask('- [z] text',3,true))
})
