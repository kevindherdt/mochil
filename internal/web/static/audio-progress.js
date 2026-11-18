import { createProgressSync } from './progress-sync.js?v=3'
import { createPassiveProgressOwnership } from './passive-progress-ownership.js?v=3'

/**
 * Where playback of an item starts. Audiobooks and podcast episodes continue where you stopped; a song always starts
 * from the beginning, because its saved position is only where it last happened to be playing.
 */
export function startPosition(item) {
  if (item?.library_kind === 'music') return 0
  const seconds = Number(item?.progress?.position_seconds || 0)
  return Number.isFinite(seconds) && seconds > 0 ? seconds : 0
}

export function createAudioProgress({ requestJSON, getActive, getAudio, storage = localStorage, cryptoObject = crypto }) {
  const deviceID = storage.getItem('mochil-audio-device') || cryptoObject.randomUUID?.() || `${Date.now()}-${Math.random()}`
  storage.setItem('mochil-audio-device', deviceID)
  let lastSaved = -1, generation = 0, claimGeneration = 0
  const ownership = createPassiveProgressOwnership()

  const endpoint = (libraryID, itemID) =>
    `/api/audio-progress/${encodeURIComponent(libraryID)}/${encodeURIComponent(itemID)}`

  const synchronization = createProgressSync({
    save: async ({ libraryID, itemID, claimGeneration: _claimGeneration, ...value }) =>
      (await requestJSON(endpoint(libraryID, itemID), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(value),
      })).data,
    resolveConflict: async (payload, _error, isCurrent) => {
      try {
        const current = (await requestJSON(endpoint(payload.libraryID, payload.itemID))).data
        if (!isCurrent()) return { handled: true }
        // Adopt the revision, but do not let an earlier sample consume or
        // suppress a seek made while its request was in flight.
        if (payload.claimGeneration !== claimGeneration) return { revision: current.revision, handled: true }
        const active = getActive()
        const audio = getAudio()
        const isActivelyPlaying = active?.library_id === payload.libraryID && active?.id === payload.itemID && !audio.paused
        const retry = ownership.resolve(current, payload, isActivelyPlaying)
        return { revision: current.revision, retry, handled: true }
      } catch {
        // A later timed save retries reconciliation without interrupting playback.
        return { handled: true }
      }
    },
  })

  function reset(item) {
    generation++
    lastSaved = -1
    ownership.reset()
    synchronization.reset(item?.progress?.revision)
  }

  function activate() {
    ownership.activate()
    lastSaved = -1
  }

  function claim() {
    claimGeneration++
    ownership.claim()
    lastSaved = -1
  }

  async function save(force = false, item = getActive(), element = getAudio()) {
    if (!item || !Number.isFinite(element.duration) || element.duration <= 0) return false
    if (!ownership.canSave(element.currentTime)) return false
    if (!force && Math.abs(element.currentTime - lastSaved) < 10) return false
    lastSaved = element.currentTime
    const currentGeneration = generation
    const currentClaimGeneration = claimGeneration
    const explicitSeek = ownership.claimed()
    try {
      const saved = await synchronization.submit({
        libraryID: item.library_id,
        itemID: item.id,
        position_seconds: element.currentTime,
        duration_seconds: element.duration,
        playback_rate: element.playbackRate,
        device_id: deviceID,
        explicit_seek: explicitSeek,
        claimGeneration: currentClaimGeneration,
      })
      if (currentGeneration !== generation) return false
      if (currentClaimGeneration === claimGeneration) {
        if (saved) ownership.releaseClaim()
        else if (explicitSeek) ownership.claim()
      }
      return saved
    } catch {
      if (currentGeneration === generation) {
        lastSaved = -1
        if (currentClaimGeneration === claimGeneration && explicitSeek) ownership.claim()
      }
      return false
    }
  }

  return { reset, activate, claim, save }
}
