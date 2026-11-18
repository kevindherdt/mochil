// Read-along for EPUBs that carry narration (EPUB 3 Media Overlays, for example
// Storyteller books). Plays the embedded audio, highlights the sentence being
// spoken and turns pages with it. It is optional: a book without narration never
// shows the control, and nothing here can stop the book from being read.
import { highlightColor, highlightCSS } from './read-along-colors.js?v=1'

const ACTIVE = 'bobo-read-along-active'
// A sentence read for longer than this is a stretch the aligner could not split: the audio is
// right but the highlighted words are not, so it is played without a highlight or tap target.
const MAX_SENTENCE = 90
const SPEEDS = [1, 1.25, 1.5, 1.75, 2, 0.85]

/** Flattens the overlay into one list of clips in reading order. */
export function flattenOverlay(overlay) {
  const clips = []
  for (const chapter of overlay?.chapters ?? []) {
    for (const [id, audio, begin, end] of chapter.clips) {
      if (end > begin) clips.push({ href: chapter.href, id, audio: overlay.audio[audio], begin, end })
    }
  }
  return clips
}

/** True for a clip too long to be one sentence. */
export const isCoarse = clip => clip.end - clip.begin > MAX_SENTENCE

/** The first clip of `href` whose text is on screen, else the first clip of that chapter. */
export function firstClipOnPage(clips, href, isOnPage) {
  const own = clips.map((clip, index) => ({ clip, index })).filter(({ clip }) => clip.href === href)
  return (own.find(({ clip }) => isOnPage(clip.id)) ?? own[0])?.index ?? -1
}

export async function loadReadAlong(url, fetchImpl = fetch) {
  try {
    const response = await fetchImpl(url, { credentials: 'same-origin' })
    if (!response.ok) return null
    const clips = flattenOverlay(await response.json())
    return clips.length ? clips : null
  } catch { return null }
}

export function createReadAlong({ clips, base, order = [], currentFrame, currentHref, goTo, button, speedButton, title, document: doc = document }) {
  const audio = new Audio()
  audio.preload = 'auto'
  let index = -1
  let playing = false
  let speed = Number(localStorage.getItem('mochil-read-along-speed')) || 1
  let loadedSource = ''

  const frameDocument = () => { try { return currentFrame()?.document ?? null } catch { return null } }
  const sourceFor = clip => new URL(clip.audio, base).href

  function ensureStyle(frame) {
    try {
      const frameDoc = frame?.document
      if (!frameDoc) return
      let style = frameDoc.getElementById('bobo-read-along-style')
      if (!style) {
        style = frameDoc.createElement('style')
        style.id = 'bobo-read-along-style'
        frameDoc.head.append(style)
      }
      const css = highlightCSS(ACTIVE, highlightColor(localStorage))
      if (style.textContent !== css) style.textContent = css
    } catch {}
  }

  const onPage = id => {
    const frame = currentFrame()
    const element = frameDocument()?.getElementById(id)
    if (!frame || !element) return false
    const box = element.getBoundingClientRect()
    return box.right > 0 && box.left < frame.innerWidth && box.bottom > 0 && box.top < frame.innerHeight
  }

  function highlight(clip) {
    const frameDoc = frameDocument()
    if (!frameDoc) return
    ensureStyle(currentFrame())
    frameDoc.querySelectorAll(`.${ACTIVE}`).forEach(element => element.classList.remove(ACTIVE))
    if (!isCoarse(clip)) frameDoc.getElementById(clip.id)?.classList.add(ACTIVE)
  }

  // Turning the page can fail or never call back; narration must not wait on it.
  const navigate = target => Promise.race([
    Promise.resolve().then(() => goTo(target)).catch(() => {}),
    new Promise(resolve => setTimeout(resolve, 1500)),
  ])

  async function show(clip) {
    if (currentHref() !== clip.href || !onPage(clip.id)) await navigate(`${clip.href}#${clip.id}`)
    highlight(clip)
  }

  function setPlaying(value) {
    playing = value
    button.setAttribute('aria-pressed', String(value))
    button.setAttribute('aria-label', value ? 'Pause read-along' : 'Play read-along')
    button.dataset.state = value ? 'playing' : 'paused'
    doc.body.classList.toggle('read-along-playing', value)
    if ('mediaSession' in navigator_()) navigator_().mediaSession.playbackState = value ? 'playing' : 'paused'
  }
  const navigator_ = () => globalThis.navigator

  /** From a page without narration (a cover, say), the first narrated text after it. */
  function nextNarrated() {
    const here = order.indexOf(currentHref())
    const found = clips.findIndex(clip => order.indexOf(clip.href) >= here)
    return found
  }

  async function play(at) {
    // Resume exactly where it stopped if that sentence is still on screen; otherwise start from this page.
    if (at === undefined && index >= 0 && clips[index].href === currentHref() && onPage(clips[index].id) && audio.src) {
      try { audio.playbackRate = speed; await audio.play(); setPlaying(true); highlight(clips[index]) } catch { setPlaying(false) }
      return
    }
    let target = at ?? firstClipOnPage(clips, currentHref(), onPage)
    if (target < 0) target = nextNarrated()
    if (target < 0) return
    await seekTo(target)
    try { await audio.play(); setPlaying(true) } catch { setPlaying(false) }
  }

  async function seekTo(next) {
    index = Math.max(0, Math.min(clips.length - 1, next))
    const clip = clips[index]
    const source = sourceFor(clip)
    if (source !== loadedSource) { loadedSource = source; audio.src = source }
    audio.playbackRate = speed
    const start = () => { audio.currentTime = clip.begin }
    if (audio.readyState >= 1) start(); else audio.addEventListener('loadedmetadata', start, { once: true })
    await show(clip)
  }

  function pause() { audio.pause(); setPlaying(false) }

  function tick() {
    if (!playing) return
    const clip = clips[index]
    if (!clip) return
    if (audio.currentTime >= clip.end - 0.01) {
      const next = clips[index + 1]
      if (!next) { pause(); return }
      index += 1
      if (sourceFor(next) !== loadedSource) {
        loadedSource = sourceFor(next)
        audio.src = loadedSource
        const start = () => { audio.currentTime = next.begin; audio.play().catch(() => {}) }
        audio.addEventListener('loadedmetadata', start, { once: true })
      } else if (Math.abs(audio.currentTime - next.begin) > 0.4) {
        audio.currentTime = next.begin
      }
      show(next)
      updateMediaSession()
    }
  }
  let frame = 0
  const loop = () => { tick(); frame = requestAnimationFrame(loop) }
  audio.addEventListener('play', () => { cancelAnimationFrame(frame); loop() })
  audio.addEventListener('pause', () => cancelAnimationFrame(frame))
  audio.addEventListener('ended', () => { tick() })
  audio.addEventListener('error', () => setPlaying(false))

  function updateMediaSession() {
    const session = navigator_().mediaSession
    if (!session || !globalThis.MediaMetadata) return
    session.metadata = new MediaMetadata({ title: title(), artist: 'Read-along' })
  }
  try {
    const session = navigator_().mediaSession
    session?.setActionHandler('play', () => play())
    session?.setActionHandler('pause', pause)
    session?.setActionHandler('previoustrack', () => skip(-1))
    session?.setActionHandler('nexttrack', () => skip(1))
  } catch {}

  /** Moves by sentences: back goes to the start of this one first. */
  function skip(direction) {
    const clip = clips[index]
    const target = direction < 0 && clip && audio.currentTime - clip.begin > 2 ? index : index + direction
    return seekTo(target).then(() => playing && audio.play().catch(() => {}))
  }

  function setSpeed(value) {
    speed = value
    audio.playbackRate = value
    try { localStorage.setItem('mochil-read-along-speed', String(value)) } catch {}
    if (speedButton) speedButton.textContent = `${value}×`
  }
  setSpeed(SPEEDS.includes(speed) ? speed : 1)

  const clipIndexByKey = new Map(clips.map((clip, position) => [`${clip.href}#${clip.id}`, position]))

  /** Tap a sentence to hear it while narrating; double-tap one to start from it. */
  function listenForTaps(frameWindow) {
    try {
      const frameDoc = frameWindow.document
      if (frameDoc.__boboReadAlong) return
      frameDoc.__boboReadAlong = true
      // The outer quarters of the page turn it; a tap there is never a request to jump.
      const inTapZone = event => event.clientX > frameWindow.innerWidth * 0.25 && event.clientX < frameWindow.innerWidth * 0.75
      // The nearest narrated sentence under the tap. A clip far longer than a sentence is a
      // misaligned stretch of audio, not a place worth jumping to.
      const find = event => {
        if (!inTapZone(event)) return -1
        for (let node = event.target; node && node.nodeType === 1; node = node.parentElement) {
          const found = node.id ? clipIndexByKey.get(`${currentHref()}#${node.id}`) : undefined
          if (found !== undefined && !isCoarse(clips[found])) return found
        }
        return -1
      }
      frameDoc.addEventListener('click', event => {
        if (!playing || event.detail > 1 || event.target.closest?.('a')) return
        const found = find(event)
        if (found >= 0 && found !== index) play(found)
      })
      frameDoc.addEventListener('dblclick', event => {
        if (playing || event.target.closest?.('a')) return
        const found = find(event)
        if (found >= 0) { frameWindow.getSelection?.()?.removeAllRanges(); play(found) }
      })
    } catch {}
  }

  // The first chapter can finish loading before the narration list arrives.
  listenForTaps(currentFrame())
  button.hidden = false
  if (speedButton) speedButton.hidden = false
  button.onclick = () => (playing ? pause() : play())
  if (speedButton) speedButton.onclick = () => setSpeed(SPEEDS[(SPEEDS.indexOf(speed) + 1) % SPEEDS.length])

  return {
    frameLoaded(frameWindow) {
      ensureStyle(frameWindow)
      listenForTaps(frameWindow)
      if (playing && clips[index]) highlight(clips[index])
    },
    pause,
    skip,
    get playing() { return playing },
    get time() { return audio.currentTime },
  }
}
