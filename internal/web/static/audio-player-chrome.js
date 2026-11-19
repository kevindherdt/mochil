const SLEEP_CHOICES = [0, 15, 30, 60]

export function nextSleepMinutes(current) {
  const index = SLEEP_CHOICES.indexOf(Number(current))
  return SLEEP_CHOICES[(Math.max(index, 0) + 1) % SLEEP_CHOICES.length]
}

/** Owns presentation-only state for the persistent audio player. */
export function createAudioPlayerChrome({ document, $, player, seek, clock, onCollapse = () => {} }) {
  let sleepTimer
  let renderedPlaying

  function updateTimeline(audio, active) {
    const candidateDuration = Number.isFinite(audio.duration) ? audio.duration : Number(active?.duration_seconds || 0)
    const duration = Number.isFinite(candidateDuration) && candidateDuration > 0 ? candidateDuration : 0
    const position = Number.isFinite(audio.currentTime) ? Math.max(0, audio.currentTime) : 0
    const fraction = duration ? Math.min(100, position / duration * 100) : 0
    const playing = !audio.paused
    $('#audio-elapsed').textContent = clock(position)
    $('#audio-duration').textContent = clock(duration)
    seek.value = Math.round(fraction * 10)
    seek.setAttribute('aria-valuetext', duration ? `${clock(position)} of ${clock(duration)}` : `${clock(position)}, duration unavailable`)
    seek.style.setProperty('--seek', `${fraction}%`)
    const play = $('#audio-play')
    player.style.setProperty('--progress', `${fraction}%`)
    if (renderedPlaying !== playing) {
      const icon = playing
        ? '<rect x="7" y="6" width="3.5" height="12" rx="1.35" fill="currentColor" stroke="none"/><rect x="13.5" y="6" width="3.5" height="12" rx="1.35" fill="currentColor" stroke="none"/>'
        : '<path d="m9 7 9 5-9 5Z"/>'
      play.classList.toggle('playing', playing)
      play.setAttribute('aria-label', playing ? 'Pause' : 'Play')
      play.querySelector('.audio-play-icon').innerHTML = icon
      $('#audio-play-large').setAttribute('aria-label', playing ? 'Pause' : 'Play')
      $('#audio-play-large').querySelector('svg').innerHTML = icon
      renderedPlaying = playing
    }
  }

  function setExpanded(expanded) {
    player.classList.toggle('expanded', expanded)
    $('#audio-expand').setAttribute('aria-expanded', expanded)
    $('#audio-collapse').hidden = !expanded
    document.body.classList.toggle('audio-expanded', expanded)
    if (expanded) player.querySelector('.audio-player-more')?.removeAttribute('open')
    else onCollapse()
  }

  function cycleSleep(audio) {
    const button = $('#audio-sleep')
    const minutes = nextSleepMinutes(button.dataset.minutes || 0)
    clearTimeout(sleepTimer)
    button.dataset.minutes = minutes
    button.textContent = minutes ? `${minutes}m` : 'Zz'
    button.setAttribute('aria-label', minutes ? `Sleep timer: ${minutes} minutes` : 'Sleep timer: off')
    if (minutes) sleepTimer = setTimeout(() => {
      button.dataset.minutes = 0
      button.textContent = 'Zz'
      button.setAttribute('aria-label', 'Sleep timer: off')
      audio.pause()
    }, minutes * 60_000)
  }

  return { cycleSleep, setExpanded, updateTimeline }
}
