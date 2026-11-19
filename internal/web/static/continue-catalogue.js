export function createContinueCatalogue({ $, requestJSON, cards, collections, libraryState, getBrowseMode, getItemLayout, abortOthers, AbortController: AbortControllerClass = AbortController }) {
  let request
  const container = $('#books')

  async function load() {
    abortOthers()
    request?.abort()
    const currentRequest = new AbortControllerClass()
    request = currentRequest
    $('#series-index').hidden = true
    $('#collection-index').hidden = true
    $('#book-catalog').hidden = false
    $('#alphabet').hidden = true
    $('#load-more').hidden = true
    $('#book-load-previous').hidden = true
    $('#book-top-spacer').style.height = '0px'
    $('#book-bottom-spacer').style.height = '0px'
    cards.releaseCovers(container)
    container.innerHTML = ''
    $('#status').textContent = 'Opening your current books…'
    try {
      const params = new URLSearchParams({ library: libraryState.selected(), limit: 100 })
      const result = await requestJSON(`/api/reading?${params}`, { signal: currentRequest.signal })
      if (request !== currentRequest || getBrowseMode() !== 'continue') return false
      const books = result.data.map(book => ({ ...book, reading_progress: { fraction: book.fraction, finished: book.finished, updated_at: book.updated_at } }))
      const layout = getItemLayout()
      container.classList.toggle('book-list', layout === 'list')
      container.innerHTML = books.length ? books.map(book => cards.markup(book, layout)).join('') : '<p class="empty-state">Books you start will appear here.</p>'
      $('#status').textContent = ''
      cards.installCoverFallbacks(container)
      collections.installActions(container)
      return true
    } catch (error) {
      if (error.name !== 'AbortError' && request === currentRequest) $('#status').textContent = `Could not open current books: ${error.message}`
      return false
    }
  }

  function abort() { request?.abort() }
  return { abort, load }
}
