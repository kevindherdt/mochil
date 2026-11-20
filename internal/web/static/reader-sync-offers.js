import { fixedLayoutPage, readingFractionsEquivalent } from '/assets/reader-remote-position.js?v=3'

// The dialogs that offer a position another device saved: the server's own
// position, or one KOReader pushed through KOSync. The reader decides nothing
// here; it only hands over what it can do (go to a place, say where it is).
export function createReaderSyncOffers({ $, view, library, id, deviceID, readingSync, resolveTextAnchor, resolveKoReaderPosition, currentReaderPosition, fixedLayoutKind, pageCount }) {
  let syncCheckInFlight = false

  // A small note at the bottom with one way back, for a jump nobody asked for.
  function announce(message, undo) {
    document.querySelector('.sync-toast')?.remove()
    const toast = Object.assign(document.createElement('div'), { className: 'sync-toast' })
    toast.setAttribute('role', 'status')
    toast.style.cssText = 'position:fixed;left:50%;bottom:24px;transform:translateX(-50%);z-index:60;display:flex;align-items:center;gap:12px;max-width:calc(100vw - 32px);padding:8px 8px 8px 16px;border:1px solid var(--line);border-radius:12px;background:var(--bg);color:var(--text);box-shadow:0 10px 30px rgb(20 16 12/.25);font:600 14px/1.25 "Source Sans","Segoe UI",sans-serif'
    toast.append(message)
    const back = Object.assign(document.createElement('button'), { type: 'button', textContent: 'Stay here' })
    back.style.cssText = 'flex:none;min-height:36px;padding:0 14px;border:0;border-radius:8px;background:var(--accent);color:var(--accent-ink,#fff);font:inherit;cursor:pointer'
    back.onclick = () => { toast.remove(); undo() }
    toast.append(back)
    document.body.append(toast)
    setTimeout(() => toast.remove(), 12000)
  }

  // Goes back to where this device was before a position was followed, and keeps it.
  async function stayAt(previous) {
    readingSync.applyingRemote = true
    try {
      if (previous?.cfi) await view.goTo(previous.cfi)
      else if (Number.isFinite(previous?.fraction)) await view.goToFraction(previous.fraction)
    } finally { readingSync.applyingRemote = false }
    $('#sync-stay').onclick()
  }

  async function offerKoSyncPosition(serverPosition, { auto = false } = {}) {
    const dialog = $('#sync-position')
    if (dialog.open) return
    const response = await fetch(`/api/books/${encodeURIComponent(library)}/${encodeURIComponent(id)}/sync`)
    if (!response.ok || response.status === 202) return
    const sync = await response.json()
    const remote = sync.remote || sync.candidate
    if (!remote || remote.device_id === deviceID || !Number.isFinite(remote.percentage)) return
    if (serverPosition?.kosync_seen_at && remote.timestamp <= serverPosition.kosync_seen_at) return
    // Following without asking is only right when the other device really is ahead in time.
    if (auto && Number(serverPosition?.updated_at) >= remote.timestamp * 1000) return
    const resolved = await resolveTextAnchor(serverPosition?.web_anchor) || await resolveKoReaderPosition(remote.progress)
    const markSeen = async accepted => {
      const response = await fetch(`/api/reading/${encodeURIComponent(library)}/${encodeURIComponent(id)}/kosync-seen`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ timestamp: remote.timestamp, ...accepted }),
      })
      if (response.ok) readingSync.revision = (await response.json()).revision
    }
    const needsLink = !sync.remote && Boolean(sync.candidate)
    const location = resolved?.accuracy === 'text' ? 'the matching text in the book'
      : resolved?.accuracy === 'exact' ? 'an exact EPUB location'
      : resolved?.accuracy === 'chapter' ? 'the start of the matching EPUB section'
        : `approximately ${Math.round(remote.percentage * 100)}%`
    $('#sync-detail').textContent = needsLink
      ? `${remote.device || 'KOReader'} recently pushed ${location} for a renamed or different file. Is this the same book?`
      : `${remote.device || 'KOReader'} has ${location}.`
    $('#sync-open').textContent = needsLink ? 'Link and go there' : 'Go there'
    $('#sync-open').onclick = async () => {
      dialog.close()
      if (needsLink) {
        const linked = await fetch(`/api/reading/${encodeURIComponent(library)}/${encodeURIComponent(id)}/kosync-link`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ document: remote.document }) })
        if (linked.ok) readingSync.revision = (await linked.json()).revision
      }
      readingSync.applyingRemote = true
      const relocated = new Promise(resolve => {
        const timeout = setTimeout(resolve, 1500)
        view.addEventListener('relocate', () => { clearTimeout(timeout); resolve() }, { once: true })
      })
      if (resolved) await view.goTo(resolved.cfi)
      else await view.goToFraction(remote.percentage)
      await relocated
      const accepted = view.lastLocation?.fraction != null
        ? { cfi: view.lastLocation.cfi || '', fraction: view.lastLocation.fraction, web_anchor: readingSync.latestAnchor }
        : null
      readingSync.applyingRemote = false
      await markSeen(accepted)
      if (accepted) {
        readingSync.acceptRemote(accepted, 'Synced from KOReader')
      }
    }
    $('#sync-stay').onclick = async () => {
      dialog.close()
      await markSeen(null)
      const position = currentReaderPosition()
      if (position) {
        readingSync.claimLocal(position)
        readingSync.save(position).catch(() => {})
      }
    }
    // Opening a book follows the other device without asking; there is always a way back.
    if (auto && !needsLink) {
      const previous = currentReaderPosition()
      await $('#sync-open').onclick()
      announce(`Continued from ${remote.device || 'KOReader'}`, () => stayAt(previous))
      return
    }
    dialog.showModal()
  }

  async function goToServerPosition(position) {
    const page = fixedLayoutPage(position, fixedLayoutKind(), pageCount())
    const anchored = await resolveTextAnchor(position?.web_anchor)
    if (page != null) await view.goTo(page)
    else if (anchored?.cfi) await view.goTo(anchored.cfi)
    else if (position?.cfi) await view.goTo(position.cfi)
    else await view.goToFraction(Number(position?.fraction) || 0)
  }

  async function offerServerPosition(position, previousRevision) {
    if (!position || Number(position.revision) <= previousRevision || !Number.isFinite(Number(position.fraction))) return false
    const current = currentReaderPosition()
    if (current && readingFractionsEquivalent(current.fraction, position.fraction)) {
      readingSync.acceptRemote(current, 'Synced')
      return false
    }
    const dialog = $('#sync-position')
    const source = position.device_name || (position.device_id === deviceID ? 'Another tab' : 'Another device')
    $('#sync-detail').textContent = `${source} is at ${Math.round(Number(position.fraction) * 100)}%.`
    $('#sync-open').textContent = 'Go there'
    $('#sync-open').onclick = async () => {
      dialog.close()
      readingSync.applyingRemote = true
      const relocated = new Promise(resolve => {
        const timeout = setTimeout(resolve, 1500)
        view.addEventListener('relocate', () => { clearTimeout(timeout); resolve() }, { once: true })
      })
      try {
        await goToServerPosition(position)
        await relocated
        const accepted = currentReaderPosition()
        if (accepted) readingSync.acceptRemote(accepted, 'Synced from another device')
      } finally {
        readingSync.applyingRemote = false
      }
    }
    $('#sync-stay').onclick = () => {
      dialog.close()
      const local = currentReaderPosition()
      if (local) {
        readingSync.claimLocal(local)
        readingSync.save(local).catch(() => {})
      }
    }
    dialog.showModal()
    return true
  }

  async function checkForRemotePosition() {
    if (syncCheckInFlight || readingSync.applyingRemote || $('#sync-position').open || document.visibilityState !== 'visible') return false
    syncCheckInFlight = true
    try {
      const previousRevision = readingSync.revision
      const serverPosition = await readingSync.load().catch(() => null)
      const offeredServerPosition = await offerServerPosition(serverPosition, previousRevision)
      if (!offeredServerPosition) await offerKoSyncPosition(serverPosition)
      return $('#sync-position').open
    } finally {
      syncCheckInFlight = false
    }
  }


  return { goToServerPosition, checkForRemotePosition, offerKoSyncPosition }
}
