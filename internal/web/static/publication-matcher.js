export function publicationSuggestionMarkup(result, index, issue, escapeHTML) {
  const authors = (result.authors || []).join(' & ')
  const value = issue.code === 'missing_author' ? authors : issue.code === 'missing_title' ? result.title : ''
  const cover = result.cover_url
    ? `<img src="/api/metadata/suggestion-artwork?url=${encodeURIComponent(result.cover_url)}" alt="Cover suggestion for ${escapeHTML(result.title)}" loading="lazy" decoding="async">`
    : ''
  return `<div class="metadata-suggestion${cover ? ' has-cover' : ''}">${cover}<span><strong>${escapeHTML(result.title)}${result.year ? ` <small>(${result.year})</small>` : ''}</strong><small>${escapeHTML(authors || 'Unknown author')} · ${escapeHTML(result.provider)}</small>${result.overview ? `<small class="metadata-result-overview">${escapeHTML(result.overview)}</small>` : ''}<small class="metadata-cover-status" role="status"></small></span><div class="metadata-suggestion-actions">${value ? `<button type="button" data-suggestion="${index}" data-value="${escapeHTML(value)}">Use ${escapeHTML(issue.code === 'missing_author' ? 'author' : 'title')}</button>` : ''}${cover ? `<button type="button" data-cover-action="save" data-result="${index}">Use cover</button>` : ''}<a href="${escapeHTML(result.source_url)}" target="_blank" rel="noopener noreferrer">Source</a></div></div>`
}

export function createPublicationMatcher({ escapeHTML, requestJSON, getContext }) {
  async function syncCoverState(target, bookID, results) {
    try {
      const { job } = getContext()
      const response = (await requestJSON(`/api/publication-metadata/${encodeURIComponent(job.library_id)}/${encodeURIComponent(bookID)}`)).data
      if (!response.configured) return
      const index = results.findIndex(result => result.cover_url === response.metadata?.artwork_url)
      const button = target.querySelector(`[data-cover-action][data-result="${index}"]`)
      if (!button) return
      button.dataset.coverAction = 'remove'
      button.textContent = 'Remove cover'
      button.closest('.metadata-suggestion').querySelector('.metadata-cover-status').textContent = 'Current catalogue cover.'
    } catch {
      // Matching remains usable when optional override state cannot load.
    }
  }

  async function updateCover(button, result, bookID) {
    const { job } = getContext()
    const status = button.closest('.metadata-suggestion').querySelector('.metadata-cover-status')
    const removing = button.dataset.coverAction === 'remove'
    button.disabled = true
    status.textContent = removing ? 'Restoring the embedded cover…' : 'Saving this cover…'
    try {
      await requestJSON(`/api/publication-metadata/${encodeURIComponent(job.library_id)}/${encodeURIComponent(bookID)}`, removing ? { method: 'DELETE' } : { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ artwork_url: result.cover_url, provider: result.provider, provider_id: result.provider_id }) })
      button.dataset.coverAction = removing ? 'save' : 'remove'
      button.textContent = removing ? 'Use cover' : 'Remove cover'
      status.textContent = removing ? 'Embedded cover restored.' : 'Cover saved as a reversible catalogue override.'
    } catch (error) {
      status.textContent = `Could not ${removing ? 'remove' : 'save'} this cover: ${error.message}`
    } finally {
      button.disabled = false
    }
  }

  async function search(article, container) {
    const { job, libraries } = getContext()
    const button = container.querySelector('[data-online-lookup]')
    const target = container.querySelector('.metadata-suggestions')
    const provider = libraries.find(library => library.id === job.library_id)?.metadata_provider
    const providerName = provider === 'hardcover' ? 'Hardcover' : 'Open Library'
    button.disabled = true
    target.textContent = `Searching ${providerName}…`
    try {
      const response = (await requestJSON(`/api/metadata/suggestions?library=${encodeURIComponent(job.library_id)}&book=${encodeURIComponent(article.dataset.book)}`)).data
      const issue = job.items.find(item => item.book_id === article.dataset.book && item.code === article.dataset.code)
      const results = response.results || []
      target.innerHTML = results.length ? `<p>Shared: <b>${escapeHTML(response.shared.title)}</b>${response.shared.author ? ` · ${escapeHTML(response.shared.author)}` : ''}</p>${results.map((result, index) => publicationSuggestionMarkup(result, index, issue, escapeHTML)).join('')}` : '<p>No close matches were returned. Keep the current value or enter a correction manually.</p>'
      target.querySelectorAll('[data-suggestion]').forEach(use => use.onclick = () => { const input = container.querySelector('input'); input.value = use.dataset.value; input.dispatchEvent(new Event('input', { bubbles: true })); input.focus() })
      target.querySelectorAll('[data-cover-action]').forEach(action => action.onclick = () => updateCover(action, results[Number(action.dataset.result)], article.dataset.book))
      await syncCoverState(target, article.dataset.book, results)
    } catch (error) {
      target.textContent = `Could not search ${providerName}: ${error.message}`
    } finally {
      button.disabled = false
    }
  }

  return { search }
}
