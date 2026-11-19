import { createBucketTracker } from './catalog-scroll-position.js?v=2'
import { createDetailRestore } from './detail-restore.js?v=3'
import { createListReturn } from './catalogue-return.js?v=1'
import { emptySpec, errorSpec, renderStatePanel } from './state-panel.js?v=2'

/** Books per page in an author's detail. */
const AUTHOR_PAGE = 120

const stateHandlers = (retry) => ({
  retry,
  settings: () => { location.hash = "#/settings"; },
  "clear-search": () => {
    const input = document.querySelector("#search");
    if (!input) return;
    input.value = "";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.focus();
  },
});

export function createAuthorCatalogue({
  $, document, escapeHTML, requestJSON, createBoundedWindow, createContinuousPager, cards, collections,
  libraryState, getBrowseMode, getItemLayout, getGroupLayout, abortOthers, getActiveBucket,
  AbortController: AbortControllerClass = AbortController,
}) {
  let request
  let authors = []
  // Offset of the first row currently in the list. Trimming moves it forward, so
  // paging back can ask for exactly the rows the window dropped.
  let windowStart = 0
  let total = 0
  let loadingEarlier = false
  let earlierPager
  const PAGE_SIZE = 80
  let loading = false
  let pager
  let authorWindow
  const bucketTracker = createBucketTracker()
  const index = $('#series-index')
  const restore = createDetailRestore()

  function installActions(search) {
    cards.installCoverFallbacks(index)
    collections.installActions(index)
    index.querySelectorAll('.author-card button[data-index]:not([data-ready]), .author-row[data-index]:not([data-ready])').forEach(button => {
      button.dataset.ready = 'true'
      button.onclick = () => open(authors[Number(button.dataset.index)], search)
    })
  }

  const listReturn = createListReturn({ count: () => authors.length, load: () => load(), loadMore: () => load(true) })

  async function open(author, search) {
    listReturn.remember()
    if (!author) return
    restore.set(author.name)
    cards.releaseCovers(index)
    const count = author.count == null ? '' : `<p>${author.count.toLocaleString()} ${author.count === 1 ? 'book' : 'books'}</p>`
    index.innerHTML = `<div class="series-focus-head"><button class="back-button" aria-label="Back to authors"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 5-7 7 7 7"/></svg></button><div><h2>${escapeHTML(author.name)}</h2>${count}</div></div><div class="series-focus-books"><p>Loading books…</p></div>`
    index.querySelector('.back-button').onclick = () => listReturn.restore()
    const target = index.querySelector('.series-focus-books')
    // An author with more books than one page used to lose the rest: the detail
    // asked for 500 and ignored the total, so books past the first page could not
    // be reached at all. Load a page at a time and offer the remainder.
    const loadedBooks = []
    let available = author.count || 0
    const paint = () => {
      const layout = getItemLayout()
      const markup = loadedBooks.map(book => cards.markup(book, layout, false)).join('')
      const remaining = available - loadedBooks.length
      const footer = remaining > 0
        ? `<div class="author-more"><button type="button" class="load-more" data-author-more>Load more books (${remaining.toLocaleString()} left)</button></div>`
        : ''
      target.innerHTML = `<div class="grid ${layout === 'list' ? 'book-list' : ''}">${markup}</div>${footer}`
      cards.installCoverFallbacks(target)
      collections.installActions(target)
      const more = target.querySelector?.('[data-author-more]')
      if (more) {
        more.onclick = () => {
          more.disabled = true
          loadPage().catch(() => { more.disabled = false })
        }
      }
    }
    const loadPage = async () => {
      const params = new URLSearchParams({ library: author.library_id, author: author.name, limit: String(AUTHOR_PAGE), offset: String(loadedBooks.length) })
      if (search) params.set('q', search)
      const result = await requestJSON(`/api/books?${params}`)
      const reported = Number(result.response?.headers?.get('X-Total-Count'))
      if (Number.isFinite(reported) && reported > 0) available = reported
      loadedBooks.push(...(result.data || []))
      paint()
      return loadedBooks.length > 0
    }
    try {
      if (!(await loadPage())) target.innerHTML = '<p>No books found for this author.</p>'
    } catch {
      target.innerHTML = '<p role="alert">Could not load this author. Please go back and try again.</p>'
    }
  }

  const listMarkup = (items, isList, start) => isList
    ? items.map((author, position) => `<button class="author-row" data-index="${start + position}" data-bucket="${escapeHTML(author.bucket || '#')}"><span><strong>${escapeHTML(author.name)}</strong>${!libraryState.selected() && author.library_name ? `<small>${escapeHTML(author.library_name)}</small>` : ''}</span><span class="series-count">${author.count.toLocaleString()} ${author.count === 1 ? 'book' : 'books'}</span></button>`).join('')
    : items.map((author, position) => {
        const context = !libraryState.selected() && author.library_name ? `<p>${escapeHTML(author.library_name)}</p>` : ''
        const countLabel = `${author.count.toLocaleString()} ${author.count === 1 ? 'book' : 'books'}`
        // Keep the count (and the shelf, when several libraries are shown)
        // in the accessible name instead of replacing it with just the title.
        const openLabel = `Open books by ${author.name}, ${countLabel}${context ? `, ${author.library_name}` : ''}`
        return `<article class="series-card author-card" data-bucket="${escapeHTML(author.bucket || '#')}"><button data-index="${start + position}" aria-label="${escapeHTML(openLabel)}"><span class="cover-frame"><img class="cover" decoding="async" data-src="${escapeHTML(author.cover_url)}" data-title="${escapeHTML(author.name)}" alt="Representative cover for ${escapeHTML(author.name)}"><span class="series-count-badge">${author.count.toLocaleString()}</span></span><h2>${escapeHTML(author.name)}</h2>${context}</button></article>`
      }).join('')

  function render(items, append, search) {
    const start = authors.length
    authors.push(...items)
    collections.syncSaveSearch?.(authors.length > 0)
    if (!authors.length) {
      renderStatePanel(index, emptySpec({ noun: 'authors', query: $('#search')?.value || '', bucket: getActiveBucket() }), stateHandlers(() => load(false)))
      return
    }
    const isList = getGroupLayout() === 'list'
    const wrapperClass = isList ? 'series-list author-list' : 'series-cover-grid author-cover-grid'
    const wrapperSelector = isList ? '.author-list' : '.author-cover-grid'
    if (!append) {
      index.innerHTML = `<div class="pagination-sentinel" data-earlier-sentinel aria-hidden="true"></div><button type="button" class="load-more" data-earlier-more hidden>Load earlier authors</button><div class="virtual-spacer" data-top-spacer aria-hidden="true"></div><div class="${wrapperClass}"></div><div class="virtual-spacer" data-bottom-spacer aria-hidden="true"></div>`
      authorWindow = createBoundedWindow({
        container: index.querySelector(wrapperSelector),
        topSpacer: index.querySelector('[data-top-spacer]'),
        bottomSpacer: index.querySelector('[data-bottom-spacer]'),
        onRemove: nodes => nodes.forEach(node => cards.releaseCovers(node)),
      })
    }
    const wrapper = index.querySelector(wrapperSelector)
    // A late page must not clobber an author list the person has since left.
    if (!wrapper) return
    if (!append) installEarlierPager()
    const insertion = authorWindow.begin('append')
    wrapper.insertAdjacentHTML('beforeend', listMarkup(items, isList, start + windowStart))
    const removed = authorWindow.finish(insertion)
    // `start` is how many rows were on screen before this page: the window can
    // only have trimmed rows that were already there, never the ones just added.
    const trimmed = Math.min(removed.start, start)
    if (trimmed > 0) {
      // The window dropped leading rows, so every remaining index shifted: redraw
      // the retained window from the array rather than leave stale indices that
      // would open the wrong author.
      authors.splice(0, trimmed)
      windowStart += trimmed
      wrapper.innerHTML = listMarkup(authors, isList, windowStart)
    }
    installActions(search)
  }

  // Reaching the top fetches the rows the window dropped, at the offset it now
  // starts at, and prepends them inside the same window.
  async function loadEarlier() {
    if (loadingEarlier || windowStart <= 0) return false
    loadingEarlier = true
    const offset = Math.max(0, windowStart - PAGE_SIZE)
    const limit = windowStart - offset
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (libraryState.selected()) params.set('library', libraryState.selected())
    if (getActiveBucket()) params.set('bucket', getActiveBucket())
    const search = $('#search').value.trim()
    if (search) params.set('q', search)
    try {
      const result = await requestJSON(`/api/authors?${params}`)
      const wrapper = index.querySelector(getGroupLayout() === 'list' ? '.author-list' : '.author-cover-grid')
      if (!wrapper) return false
      const items = result.data || []
      if (!items.length) { windowStart = 0; return false }
      const insertion = authorWindow.begin('prepend')
      wrapper.insertAdjacentHTML('afterbegin', listMarkup(items, getGroupLayout() === 'list', offset))
      const removed = authorWindow.finish(insertion)
      authors.unshift(...items)
      if (removed.end > 0) authors.splice(-removed.end, removed.end)
      windowStart = offset
      installActions(search)
      return true
    } catch (error) {
      $('#status').textContent = `Could not load earlier authors: ${error.message}`
      return false
    } finally {
      loadingEarlier = false
      earlierPager?.refresh()
      pager?.refresh()
    }
  }

  async function load(append = false) {
    if (loading && append) return false
    abortOthers()
    if (!append) request?.abort()
    const currentRequest = new AbortControllerClass()
    request = currentRequest
    loading = true
    $('#book-catalog').hidden = true
    $('#collection-index').hidden = true
    index.hidden = false
    if (!append) {
      cards.releaseCovers(index)
      index.innerHTML = ''
      authors = []
      windowStart = 0
      total = 0
      $('#status').textContent = 'Opening authors…'
    }
    const wanted = append ? '' : restore.take()
    if (!append) restore.set('')
    const params = new URLSearchParams({ limit: 80 })
    if (libraryState.selected()) params.set('library', libraryState.selected())
    if (getActiveBucket()) params.set('bucket', getActiveBucket())
    const search = $('#search').value.trim()
    if (search) params.set('q', search)
    if (append) params.set('offset', String(windowStart + authors.length))
    if (wanted) params.set('q', wanted)
    try {
      const result = await requestJSON(`/api/authors?${params}`, { signal: currentRequest.signal })
      if (getBrowseMode() !== 'authors' || request !== currentRequest) return false
      if (wanted) {
        $('#status').textContent = ''
        await open(result.data.find(item => item.name === wanted) || { library_id: libraryState.selected(), name: wanted }, '')
        return true
      }
      total = Number(result.response.headers.get('X-Total-Count')) || result.data.length
      $('#status').textContent = ''
      render(result.data, append, search)
      return true
    } catch (error) {
      if (error.status === 409 && append && request === currentRequest) return load(false)
      if (error.name !== 'AbortError' && request === currentRequest) {
        if (append) $('#status').textContent = `Could not load more authors: ${error.message}`
        else { $('#status').textContent = ''; renderStatePanel(index, errorSpec(error, 'your authors'), stateHandlers(() => load(false))) }
      }
      return false
    } finally {
      if (request === currentRequest) loading = false
      pager?.refresh()
    }
  }

  function bind() {
    pager = createContinuousPager({
      sentinel: $('#series-sentinel'), button: $('#series-load-more'), label: 'authors',
      canLoad: () => !index.hidden && getBrowseMode() === 'authors' && !loading && windowStart + authors.length < total,
      load: () => load(true),
    })
    installEarlierPager()
  }

  // The list shell is rebuilt on every fresh load, so the top pager is installed
  // against the elements that load just created.
  function installEarlierPager() {
    const sentinel = index.querySelector?.('[data-earlier-sentinel]')
    const button = index.querySelector?.('[data-earlier-more]')
    if (!sentinel || !button) return
    earlierPager?.disconnect()
    earlierPager = createContinuousPager({
      sentinel, button, label: 'earlier authors', rootMargin: '300px',
      canLoad: () => !index.hidden && getBrowseMode() === 'authors' && !loading && windowStart > 0,
      load: () => loadEarlier(),
    })
    earlierPager.refresh()
  }

  function abort() { request?.abort() }
  function updateCurrentFromScroll(top, setCurrentBucket) {
    const bucket = bucketTracker.find(index.querySelectorAll('.author-card[data-bucket], .author-row[data-bucket]'), top)
    if (bucket) setCurrentBucket(bucket)
  }
  return { abort, bind, load, open, updateCurrentFromScroll, openOnLoad: restore.openOnLoad, opened: restore.opened }
}
