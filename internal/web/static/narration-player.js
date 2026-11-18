// Speaks a book one sentence at a time. It asks for the audio of the next few
// sentences while the current one plays, marks the sentence being read, keeps
// it on screen, and carries on into the next chapter. Everything that touches
// the page or the network is passed in, so the logic can be tested on its own.
// The voice is made a little faster than it is spoken at 1×, so faster playback would run out of audio.
import { splitLead } from './narration-text.js?v=2'
import { createSleepTimer } from './narration-sleep.js?v=1'

export const narrationRates = [0.8, 1, 1.15, 1.3, 1.5]
const AHEAD = 5

// A second of silence. It is played inside the tap that starts narration, so phones allow the
// real audio later, and it is looped whenever the next sentence is not ready yet: a phone
// stops a page that is not playing anything, which would end narration at the first wait.
function silence(seconds = 1, rate = 8000) {
  const samples = seconds * rate, bytes = new Uint8Array(44 + samples * 2), view = new DataView(bytes.buffer)
  const text = (at, value) => { for (let i = 0; i < value.length; i++) bytes[at + i] = value.charCodeAt(i) }
  text(0, 'RIFF'); view.setUint32(4, 36 + samples * 2, true); text(8, 'WAVEfmt ')
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, 1, true)
  view.setUint32(24, rate, true); view.setUint32(28, rate * 2, true); view.setUint16(32, 2, true); view.setUint16(34, 16, true)
  text(36, 'data'); view.setUint32(40, samples * 2, true)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return 'data:audio/wav;base64,' + btoa(binary)
}
const SILENCE = silence()

export function createNarrationPlayer({
  audio, loadSentences, fetchAudio, mark, unmark, ensureVisible, firstOnPage, nextChapter,
  onState = () => {}, onError = () => {}, storage, mediaSession, mediaMetadata = () => null,
}) {
  let state = 'idle'
  let sentences = []
  let index = 0
  let run = 0 // changes whenever the current reading is abandoned
  let cache = new Map() // sentence index -> Promise<url>
  const ready = new Map() // sentence index -> url, once its audio has arrived
  let chained = -1 // a sentence whose audio the 'ended' handler has already started
  let controller = new AbortController()
  let wake = null // resolves the wait for the current sentence to finish
  let sleepAtChapterEnd = false
  const sleepTimer = createSleepTimer({ audio, onExpire: () => player.pause() })
  let voice = storage?.getItem('bobobooks-narration-voice') || ''
  const saved = Number(storage?.getItem('bobobooks-narration-rate'))
  let rate = narrationRates.includes(saved) ? saved : 1
  if ('preservesPitch' in audio) audio.preservesPitch = true

  const setState = next => {
    state = next
    // Waiting between sentences still counts as playing: silence is being played to keep the page alive.
    if (mediaSession) mediaSession.playbackState = next === 'playing' || next === 'loading' ? 'playing' : next === 'paused' ? 'paused' : 'none'
    onState(next)
  }

  function forgetAudio(keepFrom = Infinity) {
    for (const [at, pending] of cache) {
      if (at < keepFrom) {
        pending.then(url => URL.revokeObjectURL?.(url)).catch(() => {})
        cache.delete(at)
        ready.delete(at)
      }
    }
  }

  function audioFor(at) {
    if (!cache.has(at)) {
      const pending = fetchAudio(sentences[at].text, voice, controller.signal)
      pending.then(url => { if (cache.get(at) === pending) ready.set(at, url) }, () => cache.get(at) === pending && cache.delete(at))
      cache.set(at, pending)
    }
    return cache.get(at)
  }

  function prefetch(from) {
    for (let at = from; at < Math.min(sentences.length, from + AHEAD); at++) audioFor(at).catch(() => {})
  }

  // Plays silence until real audio replaces it.
  function holdWithSilence() {
    audio.loop = true
    audio.src = SILENCE
    audio.play?.().catch?.(() => {})
  }

  function abandon() {
    run++
    audio.loop = false
    controller.abort()
    controller = new AbortController()
    wake?.('stop')
    wake = null
    audio.pause()
    forgetAudio()
    unmark()
  }

  function startAudio(url) {
    audio.loop = false
    audio.src = url
    audio.playbackRate = rate
    return audio.play()
  }

  // When a sentence ends, the next one starts inside the same event, before anything else
  // runs. A locked phone stops the page as soon as nothing is playing, and page work such as
  // scrolling is held back, so waiting for either between sentences would end the narration.
  const finished = (token, at) => new Promise(resolve => {
    // Whichever way this wait ends (the sentence ended, errored, or was skipped), its listeners go with it. A listener
    // left behind would fire at the end of a later sentence and start audio for the wrong one.
    const release = () => { audio.removeEventListener('ended', done); audio.removeEventListener('error', failed) }
    wake = result => { wake = null; release(); resolve(result) }
    const done = () => {
      release()
      const url = token === run && at + 1 < sentences.length && !sleepAtChapterEnd ? ready.get(at + 1) : undefined
      if (url) { chained = at + 1; startAudio(url)?.catch?.(() => {}) }
      else if (token === run && !sleepAtChapterEnd) holdWithSilence() // also covers the wait for the next chapter
      wake?.('ended')
    }
    const failed = () => wake?.('error')
    audio.addEventListener('ended', done)
    audio.addEventListener('error', failed)
    if (token !== run) wake('stop')
  })

  async function reading(token) {
    try {
      while (token === run) {
        if (index >= sentences.length) {
          const more = await nextChapter()
          if (token !== run) return
          if (sleepAtChapterEnd) { stop(); return } // narration stays at the start of the next chapter
          if (!more) break
          sentences = await loadSentences()
          index = 0
          forgetAudio()
          if (!sentences.length) continue
          continue
        }
        const at = index
        let waiting
        if (chained === at) {
          // Already playing: the page catches up with it rather than the other way round.
          chained = -1
          waiting = finished(token, at)
          prefetch(at + 1)
          forgetAudio(at - 1)
          mark(sentences[at])
          ensureVisible(sentences[at]).catch(() => {})
          setState('playing')
        } else {
          chained = -1
          setState('loading')
          let url
          try { url = await audioFor(at) } catch (error) {
            if (token !== run) return
            throw error
          }
          if (token !== run) return
          if (index !== at) continue // the reader skipped while this sentence was being prepared
          prefetch(at + 1)
          forgetAudio(at - 1)
          mark(sentences[at])
          waiting = finished(token, at)
          await startAudio(url)
          if (token !== run) return
          if (index !== at) { wake?.('jump'); continue } // skipped during the start
          ensureVisible(sentences[at]).catch(() => {})
          setState('playing')
        }
        const outcome = await waiting
        if (outcome === 'stop') return
        if (outcome === 'error') throw new Error('The audio could not be played.')
        if (outcome !== 'jump') index = at + 1
      }
      if (token === run) { stop() }
    } catch (error) {
      if (token === run) {
        stop()
        onError(error)
      }
    }
  }

  async function start() {
    const token = ++run
    setState('loading')
    try {
      sentences = await loadSentences()
    } catch (error) { if (token === run) { stop(); onError(error) } return }
    if (token !== run) return
    if (!sentences.length) { setState('idle'); onError(new Error('There is no text to read here.')); return }
    index = Math.max(0, firstOnPage(sentences))
    // Speech starts as soon as the opening clause of the first sentence is ready.
    sentences.splice(index, 1, ...splitLead(sentences[index]))
    reading(token)
  }

  function stop() {
    sleepTimer.set(0)
    sleepAtChapterEnd = false
    abandon()
    setState('idle')
  }

  const player = {
    get state() { return state },
    get rate() { return rate },
    get voice() { return voice },
    toggle() {
      if (state === 'playing') return player.pause()
      if (state === 'paused') return player.resume()
      if (state === 'idle') {
        // Inside the tap: let the page play audio later, once the first sentence is ready.
        holdWithSilence()
        return start()
      }
      return stop() // pressing while it is still preparing cancels
    },
    pause() { if (state === 'playing') { audio.pause(); setState('paused') } },
    resume() { if (state === 'paused') { audio.play().then(() => setState('playing')).catch(() => {}) } },
    stop,
    /** Jumps by whole sentences while narration is on. */
    skip(delta) {
      if (state === 'idle' || !sentences.length) return
      index = Math.min(sentences.length - 1, Math.max(0, index + delta))
      audio.pause()
      wake?.('jump')
      if (state === 'paused') setState('loading')
    },
    /** Jumps to the sentence `pick` chooses from the chapter's sentences (-1 for none) while narration is on. */
    seek(pick) {
      if (state === 'idle' || !sentences.length) return false
      const at = pick(sentences)
      if (!(at >= 0)) return false
      index = Math.min(sentences.length - 1, at)
      audio.pause()
      wake?.('jump')
      if (state === 'paused') setState('loading')
      return true
    },
    /** Pauses after this many minutes (0 for never), or at 'chapter' end. */
    setSleep(mode) {
      sleepAtChapterEnd = mode === 'chapter'
      sleepTimer.set(typeof mode === 'number' ? mode : 0)
    },
    get sleepAtChapterEnd() { return sleepAtChapterEnd },
    sleepRemainingMs() { return sleepTimer.remainingMs() },
    setRate(next) {
      if (!narrationRates.includes(next)) return
      rate = next
      storage?.setItem('bobobooks-narration-rate', String(next))
      audio.playbackRate = rate
    },
    setVoice(next) {
      if (next === voice) return
      voice = next
      storage?.setItem('bobobooks-narration-voice', next)
      if (state === 'idle') return
      // Audio already prepared is in the old voice; read this sentence again in the new one.
      controller.abort()
      controller = new AbortController()
      forgetAudio()
      audio.pause()
      wake?.('jump')
    },
  }

  if (mediaSession) {
    const handlers = {
      play: player.resume, pause: player.pause, stop: player.stop,
      previoustrack: () => player.skip(-1), nexttrack: () => player.skip(1),
    }
    for (const [action, handler] of Object.entries(handlers)) {
      try { mediaSession.setActionHandler(action, handler) } catch { /* not every browser has every action */ }
    }
  }
  return player
}
