export function managedIngestBatchEntries(values) {
  return [...values].slice(0, 100).map(value => ({ source_path: String(value.source_path || '').trim(), target_path: String(value.target_path || '').trim() }))
}

export function managedIngestBatchSummary(preview) {
  const count = Number(preview?.entries?.length || 0)
  const conflicts = Number(preview?.conflicts || 0)
  return preview?.ready
    ? `${count.toLocaleString()} destinations are clear. No plan or file has been created.`
    : `${conflicts.toLocaleString()} ${conflicts === 1 ? 'file needs' : 'files need'} attention.`
}

export function createManagedIngestUI({ $, requestJSON }) {
  let transaction = null
  let pollTimer = null
  let discoveryCursor = ''
  const selected = new Map()
  let reviewedBatch = null

  function resetReview() {
    transaction = null
    $('#managed-ingest-review').hidden = true
    $('#managed-ingest-confirmation').value = ''
    $('#managed-ingest-execute').disabled = true
    $('#managed-ingest-discard').disabled = false
  }

  async function load() {
    clearTimeout(pollTimer)
    const workflow = $('#managed-ingest-workflow')
    try {
      const { data } = await requestJSON('/api/metadata/managed-ingest')
      workflow.hidden = !data.enabled
      if (!data.enabled) resetReview()
    } catch {
      workflow.hidden = true
      resetReview()
    }
  }

  async function plan() {
    const status = $('#managed-ingest-status')
    const source = $('#managed-ingest-source').value.trim()
    const target = $('#managed-ingest-destination').value.trim()
    if (transaction) {
      status.textContent = 'Cancel the current plan before reviewing another.'
      return
    }
    if (!source || !target) {
      status.textContent = 'Enter both relative paths.'
      return
    }
    await createPlan([{ source_path: source, target_path: target }], $('#managed-ingest-plan'))
  }

  async function createPlan(entries, button) {
    const status = $('#managed-ingest-status')
    if (transaction) { status.textContent = 'Cancel the current plan before reviewing another.'; return }
    button.disabled = true
    status.textContent = 'Rechecking every source and destination…'
    resetReview()
    try {
      transaction = (await requestJSON('/api/metadata/managed-ingest/plan', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ entries }),
      })).data
      $('#managed-ingest-review-paths').textContent = entries.length === 1 ? `${entries[0].source_path} → ${entries[0].target_path}` : `${entries.length.toLocaleString()} files in one atomic, no-clobber plan`
      $('#managed-ingest-confirmation').placeholder = transaction.id
      $('#managed-ingest-review').hidden = false
      $('#managed-ingest-execute').textContent = entries.length === 1 ? 'Copy verified file' : `Copy ${entries.length.toLocaleString()} verified files`
      status.textContent = `Plan ready. Transaction ${transaction.id}. No file has been copied.`
      updateBatchActions()
      $('#managed-ingest-confirmation').focus()
    } catch (error) {
      status.textContent = error.message
    } finally { button.disabled = false }
  }

  async function execute() {
    if (!transaction) return
    const button = $('#managed-ingest-execute')
    const status = $('#managed-ingest-status')
    button.disabled = true
    status.textContent = 'Copying and verifying…'
    try {
      await requestJSON(`/api/metadata/managed-ingest/${encodeURIComponent(transaction.id)}/execute`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ confirmation: $('#managed-ingest-confirmation').value }),
      })
      poll(transaction.id, 0)
    } catch (error) {
      status.textContent = error.message
      updateConfirmation()
    }
  }

  async function discover(append = false) {
    const button = append ? $('#managed-ingest-discovery-more') : $('#managed-ingest-discover')
    const status = $('#managed-ingest-discovery-status')
    if (!append) {
      discoveryCursor = ''
      $('#managed-ingest-discovery-results').replaceChildren()
    }
    button.disabled = true
    status.textContent = 'Inspecting recognized files…'
    const query = new URLSearchParams({ limit: '50' })
    const kind = $('#managed-ingest-kind').value
    if (kind) query.set('kind', kind)
    if (discoveryCursor) query.set('cursor', discoveryCursor)
    try {
      const { data } = await requestJSON(`/api/metadata/managed-ingest/discover?${query}`)
      renderDiscovery(data.items || [], append)
      discoveryCursor = data.next_cursor || ''
      $('#managed-ingest-discovery-more').hidden = !discoveryCursor
      status.textContent = data.items?.length
        ? `${data.items.length.toLocaleString()} file${data.items.length === 1 ? '' : 's'} inspected. Choose one to review its suggested path.`
        : append ? 'No more recognized files.' : 'No recognized media files are waiting in the incoming folder.'
      if (data.truncated) status.textContent += ' The safety scan limit was reached; organize the incoming folder into smaller batches.'
    } catch (error) {
      status.textContent = error.message
    } finally {
      button.disabled = false
    }
  }

  function renderDiscovery(items, append) {
    const results = $('#managed-ingest-discovery-results')
    if (!append) results.replaceChildren()
    for (const item of items) {
      const row = document.createElement('div')
      row.className = 'managed-ingest-discovery-row'
      const selectLabel = document.createElement('label')
      const checkbox = document.createElement('input')
      checkbox.type = 'checkbox'
      checkbox.checked = selected.has(item.source_path)
      checkbox.setAttribute('aria-label', `Select ${item.proposal?.title || item.source_path} for batch review`)
      checkbox.onchange = () => {
        if (checkbox.checked) {
          if (selected.size >= 100) { checkbox.checked = false; $('#managed-ingest-discovery-status').textContent = 'A batch review can contain at most 100 files.'; return }
          selected.set(item.source_path, { source_path: item.source_path, target_path: item.suggested_target, title: item.proposal?.title || item.source_path })
        } else selected.delete(item.source_path)
        invalidateBatchReview()
        updateBatchActions()
      }
      selectLabel.append(checkbox)
      const button = document.createElement('button')
      button.type = 'button'
      const identity = document.createElement('span')
      const title = document.createElement('strong')
      title.textContent = item.proposal?.title || item.source_path
      const source = document.createElement('small')
      source.textContent = item.source_path
      identity.append(title, source)
      const target = document.createElement('small')
      target.textContent = item.suggested_target
      button.append(identity, target)
      button.onclick = () => {
        if (transaction) {
          $('#managed-ingest-status').textContent = 'Cancel the current plan before choosing another file.'
          return
        }
        $('#managed-ingest-source').value = item.source_path
        $('#managed-ingest-destination').value = item.suggested_target
        $('#managed-ingest-status').textContent = 'Suggestion selected. Review both paths before creating the plan.'
        $('#managed-ingest-destination').focus()
      }
      row.append(selectLabel, button)
      results.append(row)
    }
    results.hidden = !results.childElementCount
  }

  function invalidateBatchReview() {
    reviewedBatch = null
    $('#managed-ingest-batch-review').hidden = true
    $('#managed-ingest-batch-plan').hidden = true
  }

  function updateBatchActions() {
    const count = selected.size
    $('#managed-ingest-batch-actions').hidden = count === 0
    $('#managed-ingest-batch-count').textContent = `${count.toLocaleString()} selected`
    $('#managed-ingest-batch-preview').disabled = count === 0 || Boolean(transaction)
  }

  async function previewBatch() {
    if (!selected.size || transaction) return
    const button = $('#managed-ingest-batch-preview')
    const status = $('#managed-ingest-batch-status')
    button.disabled = true
    status.textContent = 'Checking destinations…'
    try {
      reviewedBatch = (await requestJSON('/api/metadata/managed-ingest/preview', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ entries: managedIngestBatchEntries(selected.values()) }),
      })).data
      renderBatchReview(reviewedBatch)
    } catch (error) {
      status.textContent = error.message
      reviewedBatch = null
    } finally { button.disabled = false }
  }

  function renderBatchReview(preview) {
    const review = $('#managed-ingest-batch-review')
    const entries = $('#managed-ingest-batch-entries')
    const status = $('#managed-ingest-batch-status')
    entries.replaceChildren()
    for (const entry of preview.entries || []) {
      const row = document.createElement('div')
      row.className = 'managed-ingest-batch-entry'
      const identity = document.createElement('span')
      const title = document.createElement('strong')
      title.textContent = selected.get(entry.source_path)?.title || entry.source_path
      const source = document.createElement('small')
      source.textContent = entry.source_path
      identity.append(title, source)
      for (const issue of entry.issues || []) {
        const message = document.createElement('small')
        message.className = 'ingest-conflict'
        message.textContent = issue.message
        identity.append(message)
      }
      const label = document.createElement('label')
      label.append('Managed destination')
      const input = document.createElement('input')
      input.maxLength = 2048
      input.value = entry.target_path
      input.oninput = () => {
        const value = selected.get(entry.source_path)
        if (value) value.target_path = input.value.trim()
        reviewedBatch = null
        status.textContent = 'Destination changed. Review the batch again.'
        $('#managed-ingest-batch-plan').hidden = true
      }
      label.append(input)
      row.append(identity, label)
      entries.append(row)
    }
    status.textContent = managedIngestBatchSummary(preview)
    $('#managed-ingest-batch-plan').hidden = !preview.ready
    review.hidden = false
  }

  async function planBatch() {
    if (!reviewedBatch?.ready || transaction) return
    const entries = managedIngestBatchEntries(selected.values())
    await createPlan(entries, $('#managed-ingest-batch-plan'))
  }

  function clearBatch() {
    selected.clear()
    document.querySelectorAll('#managed-ingest-discovery-results input[type="checkbox"]').forEach(input => { input.checked = false })
    invalidateBatchReview()
    updateBatchActions()
  }

  async function discard() {
    if (!transaction) return
    const status = $('#managed-ingest-status')
    $('#managed-ingest-discard').disabled = true
    try {
      await requestJSON(`/api/metadata/managed-ingest/${encodeURIComponent(transaction.id)}/discard`, { method: 'POST' })
      status.textContent = 'Plan cancelled. No file was copied.'
      resetReview()
      updateBatchActions()
    } catch (error) {
      status.textContent = error.message
      $('#managed-ingest-discard').disabled = false
    }
  }

  async function poll(id, attempt) {
    const status = $('#managed-ingest-status')
    try {
      const { data } = await requestJSON(`/api/metadata/managed-ingest/${encodeURIComponent(id)}`)
      if (data.state === 'committed') {
        status.textContent = 'Verified copy added to managed storage. The next library scan will discover it.'
        resetReview()
        clearBatch()
        $('#managed-ingest-source').value = ''
        $('#managed-ingest-destination').value = ''
        return
      }
      if (['rolled_back', 'failed'].includes(data.state)) {
        status.textContent = data.error || 'The copy was not published. Review the transaction before retrying.'
        updateConfirmation()
        return
      }
      if (attempt >= 60) {
        status.textContent = 'The copy is still running. Its status remains available in Jobs.'
        updateConfirmation()
        return
      }
      pollTimer = setTimeout(() => poll(id, attempt + 1), 1000)
    } catch (error) {
      status.textContent = `Could not refresh copy status: ${error.message}`
      updateConfirmation()
    }
  }

  function updateConfirmation() {
    $('#managed-ingest-execute').disabled = !transaction || $('#managed-ingest-confirmation').value.trim() !== transaction.id
  }

  function bind() {
    $('#managed-ingest-plan').onclick = plan
    $('#managed-ingest-discover').onclick = () => discover(false)
    $('#managed-ingest-discovery-more').onclick = () => discover(true)
    $('#managed-ingest-batch-preview').onclick = previewBatch
    $('#managed-ingest-batch-plan').onclick = planBatch
    $('#managed-ingest-batch-clear').onclick = clearBatch
    $('#managed-ingest-execute').onclick = execute
    $('#managed-ingest-discard').onclick = discard
    $('#managed-ingest-confirmation').addEventListener('input', updateConfirmation)
  }

  return { bind, load }
}
