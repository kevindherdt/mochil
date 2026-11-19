/** Progressive pointer enhancement; menu actions remain the keyboard/touch path. */
export function createMusicOrderDrag({ container, rowSelector, onMove, finePointer = () => globalThis.matchMedia?.('(pointer: fine)').matches ?? false }) {
  let source = -1
  let target = null

  const rows = () => [...container.querySelectorAll(rowSelector)]
  const clearTarget = () => {
    target?.removeAttribute('data-drop-position')
    target = null
  }
  const reset = () => {
    clearTarget()
    rows().forEach(row => row.classList.remove('is-dragging'))
    source = -1
  }

  container.addEventListener('dragstart', event => {
    const row = event.target.closest(`${rowSelector}[draggable="true"]`)
    if (!row) return
    source = Number(row.dataset.orderPosition)
    row.classList.add('is-dragging')
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', String(source))
  })
  container.addEventListener('dragover', event => {
    if (source < 0) return
    const row = event.target.closest(rowSelector)
    if (!row || Number(row.dataset.orderPosition) === source) return clearTarget()
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
    const after = event.clientY >= row.getBoundingClientRect().top + row.getBoundingClientRect().height / 2
    if (target !== row) clearTarget()
    target = row
    row.dataset.dropPosition = after ? 'after' : 'before'
  })
  container.addEventListener('drop', async event => {
    if (source < 0 || !target) return reset()
    event.preventDefault()
    const from = source
    const to = Number(target.dataset.orderPosition)
    const after = target.dataset.dropPosition === 'after'
    reset()
    await onMove(from, to, after)
  })
  container.addEventListener('dragend', reset)

  return {
    sync(active) {
      reset()
      const enabled = active && finePointer()
      rows().forEach((row, index) => {
        row.draggable = enabled
        if (enabled) row.dataset.orderPosition = String(index)
        else delete row.dataset.orderPosition
      })
    },
  }
}
