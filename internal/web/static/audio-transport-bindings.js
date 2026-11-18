import { afterEnded, createBarControls } from './audio-bar-controls.js?v=3'

export function applyAudioShuffle(value, playbackOrder, buttons) {
  const enabled = playbackOrder.setShuffled(value)
  for (const button of buttons) {
    button.classList.toggle('active', enabled)
    button.setAttribute('aria-pressed', String(enabled))
  }
  return enabled
}

export function openNextAudioItem(next, prepared, open) {
  if (!next) return false
  if (prepared) open(next, true, prepared)
  else open(next, true)
  return true
}

export function clearClosedAudioSession(mediaSession) {
  if (!mediaSession) return
  mediaSession.metadata = null
  try { mediaSession.playbackState = 'none' } catch (_) {}
}

/** Owns browser transport controls and media-element lifecycle events. */
export function bindAudioTransport({
  $, primaryAudio, secondaryAudio, player, playerChrome, progress, save,
  getAudio, setAudio, getActive, setActive, music, playbackOrder,
  open, adjacent, adjacentItem, gapless, audiobookChapters, context, replayGain,
  playbackSession,
  directPlaybackEnabled = () => true,
  windowObject = window, navigatorObject = navigator,
}) {
  const setShuffle = value => applyAudioShuffle(
    value,
    playbackOrder,
    [$('#audio-player-shuffle-large'), $('#audio-bar-shuffle')],
  )
  for (const id of ['audio-player-shuffle-large', 'audio-bar-shuffle']) $(`#${id}`).onclick = () => setShuffle(!playbackOrder.shuffled)
  const barControls = createBarControls({ $, decks: [primaryAudio, secondaryAudio] })
  $('#audio-play').onclick = () => {
    const audio = getAudio()
    audio.paused ? audio.play() : audio.pause()
  }
  function previous() {
    const audio = getAudio()
    if (audio.currentTime > 5) {
      progress.claim()
      audio.currentTime = 0
    } else adjacent(-1)
  }
  for (const id of ['audio-previous', 'audio-bar-previous']) $(`#${id}`).onclick = previous
  for (const id of ['audio-next', 'audio-bar-next']) $(`#${id}`).onclick = () => adjacent(1)
  $('#audio-play-large').onclick = () => getAudio().paused ? getAudio().play() : getAudio().pause()
  $('#audio-previous-large').onclick = previous
  $('#audio-next-large').onclick = () => adjacent(1)
  $('#audio-expand').onclick = () => playerChrome.setExpanded(true)
  $('#audio-collapse').onclick = () => playerChrome.setExpanded(false)
  $('#audio-player-favorite').onclick = async () => {
    const active = getActive()
    if (!active) return
    await music.toggleFavorite(active)
    const favorite = music.isFavorite(active)
    const button = $('#audio-player-favorite')
    button.classList.toggle('active', favorite)
    button.setAttribute('aria-pressed', String(favorite))
    button.setAttribute('aria-label', favorite ? 'Remove from favorites' : 'Add to favorites')
  }
  $('#audio-seek').oninput = event => {
    const audio = getAudio()
    progress.claim()
    if (Number.isFinite(audio.duration)) audio.currentTime = Number(event.target.value) / 1000 * audio.duration
    playerChrome.updateTimeline(audio, getActive())
  }

  function bindAudioEvents(deck) {
    deck.addEventListener('timeupdate', () => {
      const audio = getAudio(), active = getActive()
      if (deck !== audio) return
      playerChrome.updateTimeline(audio, active)
      save(false)
      if (directPlaybackEnabled() && active?.library_kind === 'music' && Number.isFinite(deck.duration) && deck.duration - deck.currentTime <= 20) gapless.prepare(adjacentItem(1))
      if (active?.chapters?.length) {
        const chapter = audiobookChapters.update()
        $('#audio-context').textContent = [chapter?.title, context(active)].filter(Boolean).join(' · ')
      }
    })
    deck.addEventListener('play', () => {
      if (deck !== getAudio()) return
      progress.activate()
      playerChrome.updateTimeline(getAudio(), getActive())
    })
    deck.addEventListener('pause', () => {
      if (deck !== getAudio()) return
      playerChrome.updateTimeline(getAudio(), getActive())
      save(true)
    })
    deck.addEventListener('durationchange', () => {
      if (deck === getAudio()) playerChrome.updateTimeline(getAudio(), getActive())
    })
    deck.addEventListener('ended', () => {
      if (deck !== getAudio()) return
      const active = getActive()
      save(true, active, deck)
      const plan = afterEnded(barControls.repeatMode(), adjacentItem(1), playbackOrder.ordered()[0])
      if (plan.restart) {
        deck.currentTime = 0
        deck.play()
        return
      }
      const prepared = directPlaybackEnabled() && active?.library_kind === 'music' ? gapless.take(plan.item) : null
      openNextAudioItem(plan.item, prepared, open)
    })
  }
  bindAudioEvents(primaryAudio)
  bindAudioEvents(secondaryAudio)

  $('#audio-speed').onchange = event => {
    const audio = getAudio()
    audio.playbackRate = Number(event.target.value)
    localStorage.setItem('bobobooks-audio-speed', event.target.value)
    save(true)
  }
  $('#audio-replaygain').onchange = event => replayGain.setMode(event.target.value, getActive(), getAudio())
  const close = () => {
    save(true)
    playbackSession.close()
    gapless.close()
    setAudio(primaryAudio)
    playerChrome.setExpanded(false)
    player.hidden = true
    setActive(null)
    clearClosedAudioSession(navigatorObject.mediaSession)
    windowObject.dispatchEvent?.(new Event('bobo-player-closed'))
  }
  $('#audio-close').onclick = $('#audio-bar-close').onclick = close
  $('#audio-sleep').onclick = () => playerChrome.cycleSleep(getAudio())
  windowObject.addEventListener('pagehide', () => { if (!getAudio().paused) save(true) })

  if ('mediaSession' in navigatorObject) {
    navigatorObject.mediaSession.setActionHandler('play', () => getAudio().play())
    navigatorObject.mediaSession.setActionHandler('pause', () => getAudio().pause())
    navigatorObject.mediaSession.setActionHandler('nexttrack', () => adjacent(1))
    navigatorObject.mediaSession.setActionHandler('previoustrack', () => adjacent(-1))
    navigatorObject.mediaSession.setActionHandler('seekto', details => {
      if (details.seekTime != null) {
        progress.claim()
        getAudio().currentTime = details.seekTime
      }
    })
  }
  return { setShuffle }
}
