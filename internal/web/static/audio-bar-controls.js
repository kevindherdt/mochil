const REPEAT_MODES = ['off', 'all', 'one']
const LABELS = { off: 'Repeat off', all: 'Repeat all', one: 'Repeat this track' }

export const nextRepeatMode = mode => REPEAT_MODES[(REPEAT_MODES.indexOf(mode) + 1) % REPEAT_MODES.length]

/** What to do when a track ends: restart it (repeat one), wrap to the first track (repeat all), or carry on as usual. */
export function afterEnded(mode, next, first) {
  if (mode === 'one') return { restart: true }
  if (mode === 'all' && !next && first) return { item: first }
  return { item: next }
}

const clampVolume = value => Number.isFinite(value) ? Math.min(1, Math.max(0, value)) : 1

/** Wide-screen extras of the docked bar: repeat, a volume slider and mute. Shuffle is wired by the transport itself. */
export function createBarControls({ $, decks, storage = globalThis.localStorage }) {
  const read = key => { try { return storage.getItem(key) } catch (_) { return null } }
  const write = (key, value) => { try { storage.setItem(key, value) } catch (_) {} }
  let mode = REPEAT_MODES.includes(read('bobobooks-audio-repeat')) ? read('bobobooks-audio-repeat') : 'off'
  const stored = read('bobobooks-audio-volume')
  let volume = stored === null ? 1 : clampVolume(Number(stored))
  let muted = false
  const repeatButton = $('#audio-bar-repeat'), volumeInput = $('#audio-volume'), muteButton = $('#audio-mute')

  function paintRepeat() {
    repeatButton.dataset.mode = mode
    repeatButton.classList.toggle('active', mode !== 'off')
    repeatButton.setAttribute('aria-pressed', String(mode !== 'off'))
    repeatButton.setAttribute('aria-label', LABELS[mode])
    repeatButton.title = LABELS[mode]
  }
  function paintVolume() {
    for (const deck of decks) { deck.volume = volume; deck.muted = muted }
    const shown = muted ? 0 : volume
    volumeInput.value = String(Math.round(shown * 100))
    volumeInput.style.setProperty('--volume', `${shown * 100}%`)
    muteButton.classList.toggle('muted', muted || volume === 0)
    muteButton.setAttribute('aria-pressed', String(muted))
    muteButton.setAttribute('aria-label', muted ? 'Unmute' : 'Mute')
  }
  repeatButton.onclick = () => { mode = nextRepeatMode(mode); write('bobobooks-audio-repeat', mode); paintRepeat() }
  volumeInput.oninput = () => {
    volume = clampVolume(Number(volumeInput.value) / 100)
    muted = false
    write('bobobooks-audio-volume', String(volume))
    paintVolume()
  }
  muteButton.onclick = () => { muted = !muted; paintVolume() }
  paintRepeat()
  paintVolume()
  return { repeatMode: () => mode }
}
