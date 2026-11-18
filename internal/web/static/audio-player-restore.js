// Puts the audio player bar back after the page is reloaded (coming back from the editor, a refresh, an installed
// app that reopens): the track and what follows it come back paused, where they were. Browsers do not let a page
// start sound on its own, so the person presses play.
const KEY = 'bobobooks-audio-resume'
const MAX_AGE_MS = 6 * 60 * 60 * 1000
const FOLLOWING = 40

export function createPlayerRestore({ storage = localStorage, now = Date.now, windowObject = globalThis.window, getAudio, getActive, key, music, open }) {
  const read = () => {
    try {
      const saved = JSON.parse(storage.getItem(KEY))
      return saved?.item && now() - Number(saved.at) < MAX_AGE_MS ? saved : null
    } catch { return null }
  }
  const write = value => { try { storage.setItem(KEY, JSON.stringify(value)) } catch { /* private mode or a full disk: nothing to restore later */ } }

  // Saves the track now playing, the few after it and where the sound has got to, so a reload loses a few
  // seconds at most; once the player has been closed the saved session is dropped instead.
  let seen = false
  function touch() {
    const item = getActive(), audio = getAudio()
    if (!item) {
      if (seen) { seen = false; forget() }
      return
    }
    seen = true
    const queue = music.queue || [], at = Math.max(0, queue.findIndex(candidate => key(candidate) === key(item)))
    write({ at: now(), item, queue: queue.length ? queue.slice(at, at + FOLLOWING) : [item], position: Number.isFinite(audio?.currentTime) ? Math.floor(audio.currentTime) : 0 })
  }

  function forget() { try { storage.removeItem(KEY) } catch { /* nothing to forget */ } }

  function restore() {
    const saved = read()
    if (!saved || getActive()) return false
    music.setQueue(saved.queue?.length ? saved.queue : [saved.item])
    open(saved.item, true)
    const audio = getAudio()
    if (saved.position > 0 && audio?.addEventListener) audio.addEventListener('loadedmetadata', () => { audio.currentTime = saved.position }, { once: true })
    return true
  }

  windowObject?.addEventListener?.('pagehide', touch)
  windowObject?.addEventListener?.('bobo-player-closed', () => { seen = false; forget() })
  windowObject?.setInterval?.(touch, 10_000)
  return { touch, restore }
}

// Cheap check for the page shell, before the audio code is loaded.
export function hasSavedPlayer(storage = localStorage) {
  try { return Boolean(storage.getItem(KEY)) } catch { return false }
}
