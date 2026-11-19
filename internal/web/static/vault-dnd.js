// Drag a page onto a folder in the sidebar to move it. Mouse and pen only:
// the browser has no drag-and-drop for touch.

/** Where a page ends up when dropped on a folder ("" is the top of the vault). */
export function movedPath(pagePath, folder) {
  const name = String(pagePath).split('/').at(-1)
  return folder ? `${folder}/${name}` : name
}

export function mountVaultDrag({ root, canMove, findPage, library, requestJSON, onMoved, onError }) {
  let dragging = null
  const target = event => event.target.closest?.('[data-vault-folder], .vault-tree')
  const clear = () => root.querySelectorAll('.drop-target').forEach(node => node.classList.remove('drop-target'))

  root.addEventListener('dragstart', event => {
    const button = event.target.closest?.('.vault-page[data-document]')
    if (!button) return
    if (!canMove()) { event.preventDefault(); return }
    dragging = findPage(button.dataset.document)
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', dragging?.path || '')
    button.classList.add('dragging')
  })
  root.addEventListener('dragend', () => {
    dragging = null
    clear()
    root.querySelectorAll('.dragging').forEach(node => node.classList.remove('dragging'))
  })
  root.addEventListener('dragover', event => {
    const zone = dragging && target(event)
    if (!zone) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    clear()
    zone.classList.add('drop-target')
  })
  root.addEventListener('drop', async event => {
    const zone = dragging && target(event)
    if (!zone) return
    event.preventDefault()
    const page = dragging
    dragging = null
    clear()
    const destination = movedPath(page.path, zone.dataset.vaultFolder || '')
    if (destination === page.path) return
    try {
      const { data } = await requestJSON(page.text_url)
      await requestJSON(`/api/vaults/${encodeURIComponent(library())}/pages/${encodeURIComponent(page.id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: destination, revision: data.edit_revision }),
      })
      await onMoved(page, destination)
    } catch (error) {
      onError(error.message || 'Could not move that page.')
    }
  })
}
