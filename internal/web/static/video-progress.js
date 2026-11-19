import { createProgressSync } from './progress-sync.js?v=3'
import { createPassiveProgressOwnership } from './passive-progress-ownership.js?v=3'

export function createVideoProgress(options) {
  let session = createVideoProgressSession(options)
  return {
    reset(item) {
      // Keep the previous item's pending checkpoint and revision isolated while
      // the new player starts its own independent synchronization queue.
      session.retire()
      session = createVideoProgressSession(options)
      session.reset(item)
    },
    claim: () => session.claim(),
    save: force => session.save(force),
    activate: () => session.activate(),
  }
}

function createVideoProgressSession({ requestJSON, getActive, getVideo, deviceID }) {
  const ownership = createPassiveProgressOwnership()
  let generation = 0, claimGeneration = 0, lastSaved = -1
  let replayPending = false, replayInFlight = false, retired = false
  let replayYieldUntil = null
  const endpoint = payload => `/api/video-progress/${encodeURIComponent(payload.libraryID)}/${encodeURIComponent(payload.itemID)}`
  const sync = createProgressSync({
    save: async ({ claimGeneration: _claim, replay: _replay, afterReplay: _afterReplay, libraryID, itemID, ...value }) =>
      (await requestJSON(endpoint({ libraryID, itemID }), {
        method: 'PUT', keepalive: true,
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value),
      })).data,
    resolveConflict: async (payload, _error, isCurrent) => {
      try {
        const remote = (await requestJSON(endpoint(payload))).data
        if (!isCurrent()) return { handled: true }
        if (retired) {
          replayPending = false
          return { handled: true }
        }
        if (payload.claimGeneration !== claimGeneration) return { revision: remote?.revision, handled: true }
        if (payload.replay || payload.afterReplay) {
          // Restart is authorized only against the finished revision we opened.
          // A newer revision belongs to another action, even if it is finished.
          replayPending = false
          const item = getActive()
          const playing = item?.library_id === payload.libraryID && item?.id === payload.itemID && !getVideo().paused
          const remotePosition = Number(remote?.position_seconds)
          replayYieldUntil = !remote?.finished && Number.isFinite(remotePosition) ? remotePosition : null
          // Never retry the conflicted restart itself, even if its snapshot is ahead.
          // An active player may later pass the remote checkpoint naturally.
          ownership.resolve(remote, { ...payload, position_seconds: remote?.position_seconds }, playing)
          return { revision: remote?.revision, handled: true }
        }
        const item = getActive()
        const playing = item?.library_id === payload.libraryID && item?.id === payload.itemID && !getVideo().paused
        const passiveRetry = ownership.resolve(remote, payload, playing)
        return { revision: remote?.revision, retry: payload.explicit_seek || passiveRetry, handled: true }
      } catch { return { handled: true } }
    },
  })

  function reset(item) {
    generation++
    ownership.reset()
    lastSaved = -1
    replayPending = item?.progress?.finished === true && Number(item?.progress?.revision) > 0
    replayInFlight = false
    replayYieldUntil = null
    sync.reset(item?.progress?.revision)
  }

  function claim() {
    claimGeneration++
    ownership.claim()
    replayYieldUntil = null
    lastSaved = -1
  }

  async function save(force = false) {
    const item = getActive(), video = getVideo()
    if (!item || !Number.isFinite(video.duration) || video.duration <= 0 || !Number.isFinite(video.currentTime)) return false
    const position = video.currentTime, finished = video.ended
    const knownDuration = Number(item.duration_seconds)
    const duration = Number.isFinite(knownDuration) && knownDuration > 0 ? knownDuration : video.duration
    const currentGeneration = generation, currentClaim = claimGeneration
    const explicitSeek = ownership.claimed()
    // Merely opening a finished item at its initial position is not a replay.
    if (replayPending && position <= 0 && !explicitSeek) return false
    // A forced checkpoint must retain its snapshot after close clears the player.
    // Wait for the conditional restart before deciding whether it may be saved.
    if (replayInFlight && !force) return false
    const afterReplay = Boolean(replayInFlight)
    while (replayInFlight) {
      const started = replayInFlight
      const succeeded = await started
      if (generation !== currentGeneration || (!succeeded && !replayPending)) return false
    }
    if (replayYieldUntil !== null && position <= replayYieldUntil + 0.25) return false
    if (!ownership.canSave(position) || (!force && Math.abs(position - lastSaved) < 10)) return false
    const replay = replayPending && !explicitSeek
    let finishReplay
    if (replay) replayInFlight = new Promise(resolve => { finishReplay = resolve })
    let replaySaved = false
    lastSaved = position
    try {
      const saved = await sync.submit({
        libraryID: item.library_id, itemID: item.id,
        position_seconds: position,
        duration_seconds: duration,
        device_id: deviceID, finished, explicit_seek: explicitSeek || replay, replay,
        claimGeneration: currentClaim, afterReplay,
      })
      if (generation !== currentGeneration) return false
      if (saved) replayPending = false
      replaySaved = saved
      if (claimGeneration === currentClaim) {
        if (saved) ownership.releaseClaim()
        else if (explicitSeek) ownership.claim()
      }
      return saved
    } catch (error) {
      if (generation === currentGeneration) {
        lastSaved = -1
        if (claimGeneration === currentClaim && explicitSeek) ownership.claim()
      }
      if (force) throw error
      return false
    } finally {
      if (replay) {
        if (generation === currentGeneration) replayInFlight = false
        finishReplay(replaySaved)
      }
    }
  }

  return { reset, claim, save, activate: ownership.activate, retire: () => { retired = true } }
}
