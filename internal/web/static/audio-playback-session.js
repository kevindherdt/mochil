import { createAudioCompatibility } from './audio-compatibility.js?v=2'
import { AudioRecoveryAction, createAudioPlaybackRecovery } from './audio-playback-recovery.js?v=2'
import { worthConverting } from './playback-quality.js?v=2'

/** Owns source startup, bounded reconnects, and compatibility fallback for one audio item. */
export function createAudioPlaybackSession({ setStatus, isActive, updateTimeline, resumeGain, onMode = () => {}, onTerminal = () => {}, createCompatibility = createAudioCompatibility }) {
  const compatibility=createCompatibility({setStatus})
  const recovery=createAudioPlaybackRecovery()
  let retryTimer=0,stallTimer=0,currentAudio,lastOpen=null
  const clearTimers=()=>{clearTimeout(retryTimer);clearTimeout(stallTimer);retryTimer=stallTimer=0}
  function close(){clearTimers();compatibility.reset();if(currentAudio){currentAudio.autoplay=false;currentAudio.onerror=currentAudio.onwaiting=currentAudio.onplaying=currentAudio.oncanplay=currentAudio.onloadedmetadata=null};currentAudio=null}
  function open({identity,item,audio,resumeAt=0,quality='maximum'}){
    close();currentAudio=audio;audio.autoplay=true;recovery.transition(identity);onTerminal(false)
    const options={identity,item,audio,resumeAt,quality};lastOpen=options
    let started=false,usingCompatibility=false,triedOriginal=false
    // A converted stream is a convenience for slow connections, never a requirement: if it fails, the original file plays.
    const originalSource=audio.getAttribute?.('src')||''
    const converted=quality!=='maximum'&&Boolean(item.compatibility_url)&&worthConverting(item,quality)
    const attachOriginal=()=>{if(!active()||!originalSource)return false;triedOriginal=true;usingCompatibility=false;compatibility.reset();onMode('original',quality);setStatus('Playing the original file…');started=false;audio.src=originalSource;audio.load();return true}
    const active=()=>currentAudio===audio&&isActive(item)
    const begin=()=>{if(started||!active())return;started=true;if(resumeAt>0&&resumeAt<audio.duration-2)audio.currentTime=resumeAt;updateTimeline();resumeGain();audio.play().catch(()=>{})}
    const attachCompatibility=async()=>{if(!active())return;usingCompatibility=true;onMode('converted',quality);started=false;await compatibility.attach({item,audio,quality:quality==='maximum'?'high':quality,resumeAt,onReady:()=>{setStatus();begin()},onFatal:()=>recover(recovery.onError(Boolean(item.compatibility_url)))})}
    const recover=action=>{if(!active())return;clearTimers();resumeAt=Number.isFinite(audio.currentTime)&&audio.currentTime>0?audio.currentTime:resumeAt;if(converted&&usingCompatibility&&!triedOriginal&&attachOriginal())return;options.resumeAt=resumeAt;if(action.type===AudioRecoveryAction.Terminal){setStatus('Playback lost its connection.');onTerminal(true);return};setStatus(action.type===AudioRecoveryAction.Fallback?'Preparing compatibility playback…':`Reconnecting… ${action.attempt} of 3`);retryTimer=setTimeout(async()=>{if(!active())return;started=false;if(action.type===AudioRecoveryAction.Fallback||usingCompatibility)await attachCompatibility();else audio.load()},action.type===AudioRecoveryAction.Fallback?0:Math.min(2400,action.attempt*600))}
    onMode('original',quality)
    audio.onloadedmetadata=begin
    audio.onerror=()=>recover(recovery.onError(Boolean(item.compatibility_url)))
    audio.onwaiting=()=>{if(!active()||audio.paused)return;clearTimeout(stallTimer);stallTimer=setTimeout(()=>recover(recovery.onStall(Boolean(item.compatibility_url))),12000)}
    audio.onplaying=()=>{clearTimers();recovery.ready();setStatus()}
    audio.oncanplay=()=>clearTimeout(stallTimer)
    if(converted)attachCompatibility()
    else if(audio.readyState>=1)begin()
  }
  // Starts the same item again where it stopped, with a fresh set of reconnect attempts.
  function retry(){if(!lastOpen)return;recovery.reset();open({...lastOpen})}
  return {open,close,retry}
}
