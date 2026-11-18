import { videoTimeLabel } from './video-format.js?v=2'

const SKIP_SECONDS = 10
const IDLE_MS = 3000
const clamp = (value, low, high) => Math.min(high, Math.max(low, value))

/** The glass control bar of the video player: play, skip, seek, time, volume and full screen (of the page: Chrome refuses it on the dialog itself). It only drives the <video>. */
export function createVideoControls({ document, $, player, video, getDuration = () => video.duration }) {
  const seek = $('#video-seek'), playButtons = [$('#video-play'), $('#video-centre-play')]
  const elapsed = $('#video-elapsed'), total = $('#video-total'), volume = $('#video-volume'), mute = $('#video-mute')
  let idleTimer = 0
  const duration = () => { const value = Number(getDuration()); return Number.isFinite(value) && value > 0 ? value : 0 }
  const label = seconds => videoTimeLabel(seconds) || '0:00'

  function paintTime() {
    const length = duration(), position = clamp(video.currentTime || 0, 0, length || Infinity)
    elapsed.textContent = label(position)
    total.textContent = length ? label(length) : ''
    const fraction = length ? position / length * 100 : 0
    seek.value = String(Math.round(fraction * 10))
    seek.style.setProperty('--seek', `${fraction}%`)
    seek.setAttribute('aria-valuetext', length ? `${label(position)} of ${label(length)}` : label(position))
  }
  function paintPlaying() {
    const paused = video.paused
    player.classList.toggle('is-paused', paused)
    for (const button of playButtons) button.setAttribute('aria-label', paused ? 'Play' : 'Pause')
    wake()
  }
  function paintVolume() {
    const shown = video.muted ? 0 : video.volume
    volume.value = String(Math.round(shown * 100))
    volume.style.setProperty('--volume', `${shown * 100}%`)
    mute.classList.toggle('muted', shown === 0)
    mute.setAttribute('aria-label', video.muted ? 'Unmute' : 'Mute')
  }
  function toggle() { video.paused ? video.play?.() : video.pause() }
  function skip(seconds) {
    const length = duration()
    video.currentTime = clamp((video.currentTime || 0) + seconds, 0, length || Infinity)
  }
  function fullscreen() {
    if (document.fullscreenElement) document.exitFullscreen?.()
    else if (document.documentElement.requestFullscreen) document.documentElement.requestFullscreen().catch?.(() => video.webkitEnterFullscreen?.())
    else video.webkitEnterFullscreen?.()
  }
  function wake() {
    player.classList.remove('is-idle')
    clearTimeout(idleTimer)
    if (!video.paused) idleTimer = setTimeout(() => player.classList.add('is-idle'), IDLE_MS)
  }

  for (const button of playButtons) button.onclick = toggle
  $('#video-back').onclick = () => skip(-SKIP_SECONDS)
  $('#video-forward').onclick = () => skip(SKIP_SECONDS)
  $('#video-fullscreen').onclick = fullscreen
  mute.onclick = () => { video.muted = !video.muted; if (!video.muted && video.volume === 0) video.volume = 0.5 }
  volume.oninput = () => { video.muted = false; video.volume = clamp(Number(volume.value) / 100, 0, 1) }
  seek.oninput = () => {
    const length = duration()
    if (length) video.currentTime = Number(seek.value) / 1000 * length
  }
  video.addEventListener('click', toggle)
  for (const type of ['timeupdate', 'durationchange', 'loadedmetadata', 'seeked']) video.addEventListener(type, paintTime)
  for (const type of ['play', 'pause', 'ended', 'emptied']) video.addEventListener(type, paintPlaying)
  video.addEventListener('volumechange', paintVolume)
  for (const type of ['pointermove', 'pointerdown', 'focusin']) player.addEventListener(type, wake)
  player.addEventListener('keydown', event => {
    if (event.target.closest?.('select, details') || event.ctrlKey || event.metaKey || event.altKey) return
    const keys = {
      ' ': toggle, k: toggle, ArrowLeft: () => skip(-5), ArrowRight: () => skip(5), j: () => skip(-SKIP_SECONDS), l: () => skip(SKIP_SECONDS),
      f: fullscreen, m: () => mute.onclick(),
    }
    const action = keys[event.key]
    if (!action || (event.target.matches?.('button, input[type=range]') && event.key === ' ')) return
    event.preventDefault()
    action()
    wake()
  })
  paintTime()
  paintPlaying()
  paintVolume()
  return { paintTime }
}
