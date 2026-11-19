// Media clocks are sampled independently across clients. Treat only a
// sub-frame-scale difference as equivalent; never hide a meaningful rewind.
const DEFAULT_DRIFT_SECONDS = 0.25

export function shouldUploadPassiveProgress(remote, local, driftSeconds = DEFAULT_DRIFT_SECONDS) {
  const remotePosition = Number(remote?.position_seconds)
  const localPosition = Number(local?.position_seconds)
  const duration = Number(local?.duration_seconds)
  if (remote?.finished || !Number.isFinite(remotePosition) || !Number.isFinite(localPosition) || !Number.isFinite(duration) || duration <= 0) return false
  // Equivalent clock samples already represent the same playback point. Keep
  // the authoritative remote revision instead of creating revision churn or a
  // tiny rewind. Only genuinely newer local playback may claim a passive write.
  return Math.min(Math.max(localPosition, 0), duration) > remotePosition + driftSeconds
}

/**
 * Prevents a stale passive player from rewinding another device. An active
 * player may resume writes once it naturally catches the newer position.
 */
export function createPassiveProgressOwnership(driftSeconds = DEFAULT_DRIFT_SECONDS) {
  let superseded = false
  let yieldUntil = null
  let explicitlyClaimed = false
  let remoteFinished = false

  function reset() {
    superseded = false
    yieldUntil = null
    explicitlyClaimed = false
    remoteFinished = false
  }

  function claim() {
    superseded = false
    yieldUntil = null
    explicitlyClaimed = true
    remoteFinished = false
  }

  function claimed() {
    return explicitlyClaimed
  }

  function releaseClaim() {
    explicitlyClaimed = false
  }

  function activate() {
    if (!remoteFinished) superseded = false
  }

  function resolve(remote, local, activelyPlaying) {
    const retry = activelyPlaying && (explicitlyClaimed || shouldUploadPassiveProgress(remote, local, driftSeconds))
    explicitlyClaimed = false
    if (retry) {
      superseded = false
      yieldUntil = null
      remoteFinished = false
    } else if (remote?.finished) {
      superseded = true
      yieldUntil = null
      remoteFinished = true
    } else if (Number.isFinite(Number(remote?.position_seconds))) {
      yieldUntil = Number(remote.position_seconds)
      superseded = !activelyPlaying
      remoteFinished = false
    } else {
      superseded = true
    }
    return retry
  }

  function canSave(position) {
    if (superseded) return false
    if (!Number.isFinite(yieldUntil)) return true
    if (!Number.isFinite(Number(position)) || Number(position) + driftSeconds < yieldUntil) return false
    yieldUntil = null
    return true
  }

  return { activate, canSave, claim, claimed, releaseClaim, reset, resolve }
}
