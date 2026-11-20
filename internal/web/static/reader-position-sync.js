import { readingPositionsEquivalent } from './reading-position-equivalence.js?v=1'

const signature = position => `${position?.cfi || ''}\u0000${Number(position?.fraction).toFixed(12)}\u0000${JSON.stringify(position?.locators || {})}`

export function createReaderPositionSync({
  endpoint,
  deviceID,
  deviceName,
  positionStore,
  fetchImpl = fetch,
  renderStatus = () => {},
  isOnline = () => navigator.onLine,
  setTimer = setTimeout,
  clearTimer = clearTimeout,
  setRepeating = setInterval,
  clearRepeating = clearInterval,
  addOnlineListener = listener => addEventListener('online', listener),
  removeOnlineListener = listener => removeEventListener('online', listener),
}) {
  let revision = 0
  let latestAnchor = null
  let latestPosition = null
  let generation = 0
  let savedGeneration = 0
  let saveInFlight = null
  let saveTimer
  let retryTimer
  let retryAttempt = 0
  let blocked = false
  let blockedAtPosition = null
  let applyingRemote = false
  let repeatingTimer
  let started = false

  const normalize = position => position?.fraction == null
    ? null
    : { fraction: position.fraction, cfi: position.cfi || '', ...(position.locators ? { locators: position.locators } : {}) }

  const record = position => {
    const normalized = normalize(position)
    if (!normalized) return false
    const nextSignature = signature(normalized)
    if (nextSignature === signature(latestPosition)) return false
    latestPosition = normalized
    generation++
    // Moving on purpose after a conflict takes ownership back; until then what is
    // saved here is only a passive copy of a place the server has since replaced.
    if (blocked && readerPositionMoved(blockedAtPosition, normalized)) {
      blocked = false
      blockedAtPosition = null
    }
    positionStore.save(normalized, { conflicted: blocked })
    return true
  }

  const scheduleRetry = () => {
    clearTimer(retryTimer)
    const delays = [1000, 2000, 5000, 10000, 30000]
    retryTimer = setTimer(() => save().catch(() => {}), delays[Math.min(retryAttempt++, delays.length - 1)])
    renderStatus(isOnline() ? 'Retrying…' : 'Offline')
  }

  const save = async (position = latestPosition, keepalive = false) => {
    record(position)
    if (blocked || !latestPosition || savedGeneration >= generation) return
    if (saveInFlight) {
      await saveInFlight
      if (!blocked && savedGeneration < generation) return save(latestPosition, keepalive)
      return
    }
    clearTimer(saveTimer)
    clearTimer(retryTimer)
    const positionToSave = { ...latestPosition }
    const anchorToSave = latestAnchor ? { ...latestAnchor } : null
    const generationToSave = generation
    const baseRevision = revision
    renderStatus('Saving…')
    saveInFlight = (async () => {
      try {
        const response = await fetchImpl(endpoint, {
          method: 'PUT', keepalive, headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            cfi: positionToSave.cfi,
            fraction: positionToSave.fraction,
            device_id: deviceID,
            device_name: deviceName,
            base_revision: baseRevision,
            web_anchor: anchorToSave,
            locators: positionToSave.locators,
          }),
        })
        const saved = await response.json().catch(() => null)
        if (response.status === 409) {
          // The server's answer is authoritative, including when the record is gone: a
          // position deleted from another tab or marked unread comes back as revision 0.
          // Keeping the old revision here left this tab editing something that no longer
          // existed, so every later save conflicted and the page was never saved again.
          revision = Number(saved?.revision) || 0
          if (!revision) {
            blocked = false
            blockedAtPosition = null
            return 'retry-now'
          }
          blocked = true
          blockedAtPosition = { ...latestPosition }
          positionStore.markConflicted?.()
          renderStatus('Changed elsewhere')
          return 'blocked'
        }
        if (!response.ok || !saved) throw new Error(`save returned HTTP ${response.status}`)
        revision = saved.revision || revision
        // The server may idempotently acknowledge a newer cross-device value
        // for the same visible page. Do not retry it as an unacknowledged save.
        const acknowledged = readingPositionsEquivalent(saved, { ...positionToSave, web_anchor: anchorToSave })
        if (!acknowledged) throw new Error('server did not acknowledge the requested position')
        savedGeneration = Math.max(savedGeneration, generationToSave)
        retryAttempt = 0
        positionStore.markSynced(positionToSave)
        renderStatus(savedGeneration < generation ? 'Saving…' : 'Saved')
        return 'saved'
      } catch {
        scheduleRetry()
        return 'retry-later'
      }
    })()
    const outcome = await saveInFlight
    saveInFlight = null
    if (outcome === 'retry-now' || (outcome === 'saved' && savedGeneration < generation)) {
      return save(latestPosition, keepalive)
    }
  }

  const schedule = position => {
    if (!normalize(position)) return
    record(position)
    if (savedGeneration >= generation) return
    renderStatus('Saving…')
    clearTimer(saveTimer)
    saveTimer = setTimer(() => save().catch(() => {}), 400)
  }

  const load = async () => {
    const response = await fetchImpl(endpoint)
    if (response.status === 404) {
      // No position is a state this tab can write from, not a revision to keep. Holding
      // the old revision made the next save conflict with a record that is not there.
      revision = 0
      return null
    }
    if (!response.ok) throw new Error('Reading position is unavailable')
    const position = await response.json()
    revision = position.revision || 0
    return position
  }

  const flushPending = () => {
    if (!blocked && savedGeneration < generation) save().catch(() => {})
  }
  const onlineListener = () => flushPending()

  return {
    load,
    save,
    schedule,
    cancelScheduled() { clearTimer(saveTimer) },
    start() {
      if (started) return
      started = true
      repeatingTimer = setRepeating(flushPending, 5000)
      addOnlineListener(onlineListener)
    },
    stop() {
      if (!started) return
      started = false
      clearRepeating(repeatingTimer)
      clearTimer(saveTimer)
      clearTimer(retryTimer)
      removeOnlineListener(onlineListener)
    },
    acceptRemote(position, status = '') {
      const normalized = normalize(position)
      if (!normalized) return
      blocked = false
      blockedAtPosition = null
      retryAttempt = 0
      clearTimer(retryTimer)
      latestPosition = normalized
      generation++
      savedGeneration = generation
      positionStore.save(normalized)
      positionStore.markSynced(normalized)
      if (status) renderStatus(status)
    },
    claimLocal(position) {
      const normalized = normalize(position)
      if (!normalized) return false
      blocked = false
      blockedAtPosition = null
      latestPosition = normalized
      generation++
      positionStore.save(normalized)
      return true
    },
    get revision() { return revision },
    set revision(value) { revision = Number(value) || revision },
    get latestPosition() { return latestPosition },
    get latestAnchor() { return latestAnchor },
    // An anchor describes one place, so a move without a fresh anchor drops the old one.
    set latestAnchor(value) { latestAnchor = value || null },
    get applyingRemote() { return applyingRemote },
    set applyingRemote(value) { applyingRemote = Boolean(value) },
    get blocked() { return blocked },
  }
}

export function readerPositionMoved(previous, current) {
  if (!previous || !current) return false
  const previousPage = exactFixedLayoutPage(previous)
  const currentPage = exactFixedLayoutPage(current)
  if (previousPage != null || currentPage != null) return previousPage != null && currentPage != null && previousPage !== currentPage
  const left = Number(previous.fraction)
  const right = Number(current.fraction)
  return Number.isFinite(left) && Number.isFinite(right) && Math.abs(left - right) > 0.00001
}

function exactFixedLayoutPage(position) {
  for (const kind of ['comic', 'pdf']) {
    const encoded = position?.locators?.[kind]
    if (!encoded) continue
    try {
      const page = Number(JSON.parse(encoded)?.page)
      if (Number.isInteger(page) && page >= 0) return page
    } catch {}
  }
  return null
}
