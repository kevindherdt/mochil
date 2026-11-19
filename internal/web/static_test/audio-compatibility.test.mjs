import assert from 'node:assert/strict'
import test from 'node:test'

globalThis.window = {}
const { createAudioCompatibility } = await import('../static/audio-compatibility.js')

test('attaches bounded HLS fallback and tears it down', async () => {
  const events={MEDIA_ATTACHED:'attached',MANIFEST_PARSED:'parsed',ERROR:'error'}
  class Hls { static Events=events;static isSupported(){return true}constructor(options){this.options=options;this.handlers={};Hls.instance=this}on(name,fn){this.handlers[name]=fn}attachMedia(audio){this.audio=audio}loadSource(source){this.source=source}destroy(){this.destroyed=true} }
  const statuses=[],audio={removed:false,loads:0,removeAttribute(){this.removed=true},load(){this.loads++},canPlayType(){return ''}}
  let ready=false
  const compatibility=createAudioCompatibility({loadClient:async()=>Hls,setStatus:value=>statuses.push(value)})
  assert.equal(await compatibility.attach({item:{compatibility_url:'/compat',revision:'one'},audio,resumeAt:42,onReady:()=>{ready=true}}),true)
  assert.equal(Hls.instance.options.startPosition,42);assert.equal(audio.removed,true)
  Hls.instance.handlers.attached();assert.equal(Hls.instance.source,'/compat?v=one')
  Hls.instance.handlers.parsed();assert.equal(ready,true);assert.equal(statuses[0],'Preparing compatibility playback…')
  compatibility.reset();assert.equal(Hls.instance.destroyed,true)
})

test('uses native HLS when the client is unavailable', async()=>{
  const audio={src:'',loads:0,canPlayType:()=> 'maybe',load(){this.loads++}}
  const compatibility=createAudioCompatibility({loadClient:async()=>{throw new Error('offline')}})
  assert.equal(await compatibility.attach({item:{compatibility_url:'/native',revision:'r 1'},audio}),true)
  assert.equal(audio.src,'/native?v=r%201');assert.equal(audio.loads,1)
})
