import {vaultMutationCommitted} from './vault-mutation-error.js'

export function changeVaultTask(source, offset, checked) {
  if (!Number.isSafeInteger(offset) || offset < 1 || offset >= source.length - 1 ||
      source[offset - 1] !== '[' || source[offset + 1] !== ']' || !/[ xX]/.test(source[offset])) throw new Error('This task changed. Reopen the page before trying again.')
  return source.slice(0, offset) + (checked ? 'x' : ' ') + source.slice(offset + 1)
}

/** One operation per mounted page; authoritative reload supplies the next revision. */
export function bindVaultTasks({pane, item, content, revision, writable, requestJSON, isCurrent, onSaved}) {
  if (!writable) return
  const article = pane.querySelector('article')
  if (!article) return
  let busy = false
  article.addEventListener('change', async event => {
    const checkbox = event.target.closest('input[data-vault-task]')
    if (!checkbox || !article.contains(checkbox) || busy || !isCurrent()) return
    busy = true
    const checked = checkbox.checked, offset = Number(checkbox.dataset.vaultTask)
    const controls = [...article.querySelectorAll('input[data-vault-task]'), ...pane.querySelectorAll('[data-vault-edit]')]
    controls.forEach(control => { control.disabled = true })
    let status = article.querySelector('.vault-task-status')
    if (!status) { status = document.createElement('p'); status.className = 'vault-task-status'; status.setAttribute('role', 'status'); article.append(status) }
    status.textContent = 'Saving task…'
    let committed = false
    try {
      const updated = changeVaultTask(content, offset, checked)
      await requestJSON(`/api/vaults/${encodeURIComponent(item.library_id)}/pages/${encodeURIComponent(item.id)}`, {
        method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({revision, content:updated}),
      })
      committed = true
      if (!isCurrent() || !article.isConnected) return
      status.textContent = 'Saved'
      await onSaved(offset)
    } catch (error) {
      if (!isCurrent() || !article.isConnected) return
      if (committed || vaultMutationCommitted(error)) {
        status.textContent = 'Saved. Reopen the page before changing another task.'
        return
      }
      checkbox.checked = !checked
      status.textContent = error.message
      // Revision conflicts need a fresh page, not another stale write.
      if (error.status === 409 || error.status === 412) return
      controls.forEach(control => { control.disabled = false })
      busy = false
    }
  })
}
