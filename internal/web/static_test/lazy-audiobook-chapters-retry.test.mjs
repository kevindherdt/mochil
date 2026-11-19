import test from 'node:test'
import assert from 'node:assert/strict'
import {createLazyAudiobookChapters} from '../static/lazy-audiobook-chapters.js'

const tick=()=>new Promise(resolve=>setImmediate(resolve))
const item=id=>({id,library_kind:'audiobooks',chapters:[{}]})

test('chapter loading reports current failure and retries on a later open',async()=>{
  let calls=0
  const errors=[],opened=[]
  const loader=createLazyAudiobookChapters({onError:error=>errors.push(error)},async()=>{
    if(++calls===1)throw new Error('offline')
    return {createAudiobookChapters:()=>({openItem:value=>opened.push(value.id)})}
  })
  loader.openItem(item('first'))
  await tick()
  assert.equal(errors.length,1)
  assert.match(errors[0],/Reopen this audiobook/)
  loader.openItem(item('retry'))
  await tick()
  assert.equal(calls,2)
  assert.deepEqual(opened,['retry'])
})

test('shared loading mounts once and only opens the newest chaptered item',async()=>{
  let resolve,calls=0
  const pending=new Promise(done=>resolve=done),opened=[]
  const loader=createLazyAudiobookChapters({},()=>{calls++;return pending})
  loader.openItem(item('old'));loader.openItem(item('new'))
  await tick()
  resolve({createAudiobookChapters:()=>({openItem:value=>opened.push(value.id)})})
  await tick()
  assert.equal(calls,1)
  assert.deepEqual(opened,['new'])
})

test('closing or switching to music suppresses an obsolete chapter load failure',async()=>{
  for(const close of [loader=>loader.close(),loader=>loader.openItem({library_kind:'music'})]) {
    let reject
    const pending=new Promise((_,fail)=>reject=fail),errors=[]
    const loader=createLazyAudiobookChapters({onError:error=>errors.push(error)},()=>pending)
    loader.openItem(item('old'))
    await tick()
    close(loader)
    reject(new Error('offline'))
    await tick()
    assert.deepEqual(errors,[])
  }
})
