const POSITION_EQUIVALENCE = 0.002

export function selectInitialReadingPosition(local, remote) {
  const localUpdatedAt = Number(local?.updatedAt) * 1000
  const remoteUpdatedAt = Number(remote?.updated_at)
  // A copy kept after the server refused it as out of date is not unsynced reading:
  // another device has written since, so the server's place is the one to resume.
  const localPendingIsNewer = Boolean(local?.pending && !local.conflicted && Number.isFinite(localUpdatedAt) && localUpdatedAt > remoteUpdatedAt)
  // Equivalent fractions can render as adjacent pages on different viewport
  // sizes. Keep this browser's exact CFI/page while acknowledging the newer
  // server revision instead of visibly moving backward or echoing a write.
  if (local && readingFractionsEquivalent(local.fraction, remote?.fraction)) {
    return { position: local, source: 'equivalent' }
  }
  if (localPendingIsNewer) return { position: local, source: 'local' }
  if (Number.isFinite(Number(remote?.fraction))) return { position: remote, source: 'remote' }
  return { position: local, source: 'local' }
}

export function readingFractionsEquivalent(left, right) {
  return Number.isFinite(Number(left)) && Number.isFinite(Number(right))
    && Math.abs(Number(left) - Number(right)) <= POSITION_EQUIVALENCE
}

export function fixedLayoutPage(position, kind, pageCount) {
  const count = Math.trunc(Number(pageCount))
  if (!position?.locators || !kind || !Number.isFinite(count) || count < 1) return null
  const candidates = [position.locators[kind]]
  if (kind === 'comic') candidates.push(position.locators.pdf)
  for (const encoded of candidates) {
    try {
      const locator = JSON.parse(encoded)
      const page = Math.trunc(Number(locator?.page))
      if (Number.isFinite(page) && page >= 0 && page < count) return page
    } catch {}
  }
  return null
}

export function positionFromReaderLocation(view, location, kind, pageCount) {
  if (!Number.isFinite(Number(location?.fraction))) return null
  const position = { fraction: Number(location.fraction), cfi: location.cfi || '' }
  if (kind !== 'comic') return position
  const count = Math.trunc(Number(pageCount))
  const page = Number.isFinite(count) && count > 0
    ? view.resolveCFI?.(location.cfi)?.index
    : null
  if (Number.isInteger(page) && page >= 0 && page < count) {
    position.locators = { comic: JSON.stringify({ version: 1, page, page_count: count }) }
  }
  return position
}
