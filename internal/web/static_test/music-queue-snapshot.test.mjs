import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

const tick = () => new Promise(resolve => setImmediate(resolve))
const track = id => ({library_id:'music',id})
function fixture() {
  const reads=[],writes=[]
  const state=createMusicState({key:item=>`${item.library_id}:${item.id}`,onChanged(){},requestJSON:(path,options={})=>{
    assert.equal(path,'/api/audio-queue')
    if(options.method){writes.push(JSON.parse(options.body));return Promise.resolve({data:null})}
    return new Promise(resolve=>reads.push(resolve))
  }})
  return {state,reads,writes}
}

test('queue append captures selection while initial queue loads',async()=>{
  const {state,reads,writes}=fixture()
  const selected=track('chosen')
  const adding=state.addToQueue(selected)
  selected.id='different';selected.library_id='other'
  reads[0]({data:[track('existing')]});await adding
  assert.deepEqual(state.queue,[track('existing'),track('chosen')])
  assert.deepEqual(writes[0].map(item=>item.item_id),['existing','chosen'])
})

test('play next captures selected track and current anchor while initial queue loads',async()=>{
  const {state,reads}=fixture()
  const selected=track('chosen'),current=track('playing')
  const adding=state.playNext(selected,current)
  selected.id='different';current.id='last'
  reads[0]({data:[track('playing'),track('last')]});await adding
  assert.deepEqual(state.queue.map(item=>item.id),['playing','chosen','last'])
})

test('queue retains its submitted identities after caller mutations and later reorder',async()=>{
  const {state,writes}=fixture()
  const first=track('first'),second=track('second'),items=[first,second]
  state.setQueue(items)
  first.id='changed';items.reverse();items.push(track('third'))
  state.moveInQueue(track('second'),-1)
  await tick()
  assert.deepEqual(state.queue,[track('second'),track('first')])
  assert.deepEqual(writes.at(-1).map(item=>item.item_id),['second','first'])
})

test('simultaneous appends share initial read and retain both selected tracks',async()=>{
  const {state,reads}=fixture()
  const first=state.addToQueue(track('one')),second=state.addToQueue(track('two'))
  assert.equal(reads.length,1)
  reads[0]({data:[track('existing')]})
  await Promise.all([first,second])
  assert.deepEqual(state.queue.map(item=>item.id),['existing','one','two'])
})
