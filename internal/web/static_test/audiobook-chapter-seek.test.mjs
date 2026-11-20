import test from 'node:test'
import assert from 'node:assert/strict'
import {createAudiobookChapters} from '../static/audiobook-chapters.js'
import {bindAudioTransport} from '../static/audio-transport-bindings.js'

test('chapter lookup and explicit seeking follow the active audio deck',()=>{
  const calls=[]
  const primary={currentTime:10,play(){assert.fail('inactive deck played')}}
  const secondary={currentTime:75,play(){calls.push('play');return Promise.resolve()}}
  let deck=primary
  const elements={
    '#audio-chapters':{hidden:true,contains:()=>false,addEventListener(){}},
    '#audio-chapters-toggle':{setAttribute(){}},
    '#audio-chapters-close':{},
    '#audio-chapter-list':{querySelectorAll:()=>[],querySelector:()=>null},
  }
  const item={chapters:[{start_seconds:0,end_seconds:60},{start_seconds:60,end_seconds:120}]}
  const controller=createAudiobookChapters({
    $:s=>elements[s],audio:primary,getAudio:()=>deck,getActive:()=>item,
    claimSeek(){calls.push(['claim',deck.currentTime])},
    player:{ownerDocument:{activeElement:null}},updateTimeline(){calls.push('timeline')},
  })
  assert.equal(controller.current(),item.chapters[0])
  deck=secondary
  assert.equal(controller.current(),item.chapters[1])
  elements['#audio-chapter-list'].onclick({target:{closest:()=>({dataset:{chapter:'0'}})}})
  assert.equal(primary.currentTime,10)
  assert.equal(secondary.currentTime,0)
  assert.deepEqual(calls,[['claim',75],'timeline','play'])
  calls.length=0
  elements['#audio-chapter-list'].onclick({target:{closest:()=>null}})
  assert.deepEqual(calls,[])
})

test('both previous controls claim restarting the current item but not track navigation',()=>{
  const elements=new Map(),events=[]
  const $=s=>{if(!elements.has(s))elements.set(s,{dataset:{},style:{setProperty(){}},classList:{toggle(){}},setAttribute(){}});return elements.get(s)}
  const deck={currentTime:30,addEventListener(){}}
  bindAudioTransport({$,primaryAudio:deck,secondaryAudio:{addEventListener(){}},
    playerChrome:{},getAudio:()=>deck,progress:{claim(){events.push(['claim',deck.currentTime])}},
    adjacent:direction=>events.push(['adjacent',direction]),windowObject:{addEventListener(){}},navigatorObject:{},
  })
  for(const selector of ['#audio-previous','#audio-previous-large']) {
    deck.currentTime=30
    $(selector).onclick()
    assert.equal(deck.currentTime,0)
    $(selector).onclick()
  }
  assert.deepEqual(events,[['claim',30],['adjacent',-1],['claim',30],['adjacent',-1]])
})
