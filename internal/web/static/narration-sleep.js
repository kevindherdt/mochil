// A sleep timer for narration: after the chosen time the voice fades out over a
// few seconds and narration pauses where it stopped, so it can carry on from there.
// The clock and the scheduler are passed in so it can be tested without waiting.
export function createSleepTimer({ audio, onExpire, now = Date.now, setTimer = setTimeout, clearTimer = clearTimeout, fadeMs = 10000, tickMs = 500 }) {
  let deadline = 0
  let handle = null

  const restoreVolume = () => { audio.volume = 1 }

  function tick() {
    const left = deadline - now()
    if (left <= 0) {
      handle = null
      deadline = 0
      onExpire()
      restoreVolume() // the next time narration plays, it plays at full volume
      return
    }
    if (left <= fadeMs) audio.volume = Math.max(0, left / fadeMs)
    handle = setTimer(tick, tickMs)
  }

  return {
    /** Starts (or restarts) the timer; 0 turns it off. */
    set(minutes) {
      clearTimer(handle)
      handle = null
      restoreVolume()
      if (!(minutes > 0)) { deadline = 0; return }
      deadline = now() + minutes * 60000
      handle = setTimer(tick, tickMs)
    },
    /** Milliseconds until narration pauses, or 0 when no timer is running. */
    remainingMs() { return deadline ? Math.max(0, deadline - now()) : 0 },
  }
}
