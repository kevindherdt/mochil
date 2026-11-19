const escapeHTML = (documentRef, value) => {
  const span = documentRef.createElement('span')
  span.textContent = String(value || '')
  return span.innerHTML
}

export function createReaderPositionHistory({ $, endpoint, deviceID, readingSync, goToPosition, currentPosition, renderLocation, fetchImpl = fetch, documentRef = document }) {
  const describe = position => ({
    percentage: `${Math.round(Number(position.fraction || 0) * 100)}%`,
    device: position.device_name || (position.device_id === deviceID ? 'This browser' : position.device_id) || 'Unknown device',
    timestamp: new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(position.updated_at)),
  })

  const confirmRestore = (position, description) => {
    const dialog = $('#restore-position')
    $('#restore-detail').textContent = `Return to ${description.percentage} from ${description.device}, saved ${description.timestamp}?`
    $('#restore-confirm').onclick = async () => {
      const button = $('#restore-confirm')
      button.disabled = true
      try {
        const response = await fetchImpl(`${endpoint}/history`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ updated_at: position.updated_at }),
        })
        if (!response.ok) throw new Error('That position could not be restored')
        const restored = await response.json()
        readingSync.revision = restored.revision
        readingSync.applyingRemote = true
        await goToPosition(restored)
        const accepted = currentPosition()
        if (accepted) readingSync.acceptRemote(accepted)
        dialog.close()
        $('#position-history').close()
        $('#reading-options').close()
        renderLocation('Restored')
      } catch (error) {
        $('#restore-detail').textContent = `${error.message}. Try again.`
      } finally {
        button.disabled = false
        readingSync.applyingRemote = false
      }
    }
    dialog.showModal()
  }

  const open = async () => {
    const dialog = $('#position-history')
    const list = $('#history-list')
    list.innerHTML = '<p class="history-empty" role="status">Loading recent positions…</p>'
    dialog.showModal()
    try {
      const response = await fetchImpl(`${endpoint}/history`)
      if (!response.ok) throw new Error('History is unavailable')
      const positions = await response.json()
      list.innerHTML = ''
      if (!positions.length) return void (list.innerHTML = '<p class="history-empty">No earlier positions have been saved yet.</p>')
      for (const position of positions) {
        const description = describe(position)
        const entry = documentRef.createElement('div')
        const isCurrent = position.revision === readingSync.revision
        entry.className = 'history-entry'
        entry.innerHTML = `<strong>${description.percentage} · ${escapeHTML(documentRef, description.device)}</strong><small>${escapeHTML(documentRef, description.timestamp)}</small><button type="button"${isCurrent ? ' disabled' : ''}>${isCurrent ? 'Current' : 'Restore'}</button>`
        if (!isCurrent) entry.querySelector('button').onclick = () => confirmRestore(position, description)
        list.append(entry)
      }
    } catch (error) {
      list.innerHTML = `<p class="history-empty">${escapeHTML(documentRef, error.message)}. Close this view and try again.</p>`
    }
  }

  return { open }
}
