import { mountCollectionDialog } from './collection-dialog-template.js?v=4'
import { describeRule, fillRule, readRule } from './smart-collection-form.js?v=2'
import { attachSuggestions } from './field-suggestions.js?v=3'

export function createCollections({ document, $, $$, escapeHTML, requestJSON, bookMarkup, installCoverFallbacks, getSelectedLibrary, getItemLayout, isActive, abortBrowse, createBoundedWindow, createContinuousPager, releaseCovers = () => {} }) {
mountCollectionDialog(document)
let collections = []
let pendingMembers = []
let pendingRule = null // set while the dialog is saving a search instead of picking books

function installCollectionActions(root = document) {
  root.querySelectorAll('.collection-save:not([data-ready])').forEach(button => {
    button.dataset.ready = 'true'
    button.onclick = event => {
      event.preventDefault(); event.stopPropagation()
      const member = { type: button.dataset.type, library_id: button.dataset.library }
      if (member.type === 'book') member.book_id = button.dataset.book
      else member.series = button.dataset.series
      pendingMembers = [member]
      pendingRule = null
      openCollectionDialog()
    }
  })
}

const sameCollectionMember = (a, b) => a.type === b.type && a.library_id === b.library_id && (a.book_id || '') === (b.book_id || '') && (a.series || '') === (b.series || '')
async function refreshCollections(libraryID = pendingMembers[0]?.library_id || getSelectedLibrary()) { collections = (await requestJSON(`/api/collections?library=${encodeURIComponent(libraryID)}`)).data }
function renderCollectionChoices() {
  const pickable = pendingRule ? [] : collections.filter(c => !c.rule)
  $('#collection-choices').innerHTML = pickable.length ? pickable.map(c => { const saved = pendingMembers.length > 0 && pendingMembers.every(member => c.members.some(m => sameCollectionMember(m, member))); return `<button type="button" class="collection-choice" data-id="${escapeHTML(c.id)}" data-saved="${saved}" aria-pressed="${saved}"><span class="collection-choice-copy"><strong>${escapeHTML(c.name)}</strong><small>${c.count} ${c.count === 1 ? 'book' : 'books'}</small></span><span class="collection-choice-state">${saved ? 'Saved' : 'Add'}</span></button>` }).join('') : (pendingRule ? '' : '<p>No collections yet.</p>')
  $$('.collection-choice').forEach(button => button.onclick = async () => {
    const removing = button.dataset.saved === 'true'
    const state = button.querySelector('.collection-choice-state')
    button.disabled = true
    state.textContent = removing ? 'Removing…' : 'Adding…'
    $('#collection-dialog-status').textContent = ''
    try {
      const members = removing ? pendingMembers : pendingMembers.filter(member => !collections.find(c => c.id === button.dataset.id)?.members.some(m => sameCollectionMember(m, member)))
      for (const member of members) {
        const response = await fetch(`/api/collections/${encodeURIComponent(button.dataset.id)}/members`, { method: removing ? 'DELETE' : 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(member) })
        if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`)
      }
      await refreshCollections()
      renderCollectionChoices()
      $('#collection-dialog-status').textContent = removing ? 'Removed from collection.' : pendingMembers.length > 1 ? `Saved ${pendingMembers.length} books to the collection.` : 'Saved to collection.'
      if (isActive()) loadCollections()
    } catch (error) {
      button.disabled = false
      state.textContent = removing ? 'Saved' : 'Add'
      $('#collection-dialog-status').textContent = `Could not update collection: ${error.message}`
    }
  })
}
async function openCollectionDialog() {
  document.querySelector('#collection-dialog h2').textContent = pendingRule ? 'Smart collection' : 'Save to collection'
  $('#create-collection').textContent = pendingRule ? 'Save' : 'Create'
  $('#collection-rule-fields').hidden = !pendingRule
  if (pendingRule) fillRule($('#collection-dialog'), pendingRule)
  $('#collection-dialog-status').textContent = 'Loading…'; $('#collection-dialog').showModal()
  try { await refreshCollections(); renderCollectionChoices(); $('#collection-dialog-status').textContent = '' } catch (error) { $('#collection-dialog-status').textContent = error.message }
}
// One card per collection, in the series-card style: the first cover in front and, for several books, a stack behind it.
function collectionCardHTML(c) {
  const covers = c.covers || [], count = Number(c.count) || 0
  const stack = count > 1 && covers.length ? ` is-stack" style="--stack-cover:url('${escapeHTML(encodeURI(covers[1] || covers[0]))}')` : ''
  const detail = `${c.rule ? `${describeRule(c.rule)} · ` : ''}${count} ${count === 1 ? 'book' : 'books'}`
  return `<article class="series-card collection-card${stack}"><button data-id="${escapeHTML(c.id)}" aria-label="Open ${escapeHTML(c.name)}"><span class="cover-frame"><img class="cover" decoding="async" data-src="${escapeHTML(covers[0] || '')}" data-title="${escapeHTML(c.name)}" alt=""><span class="series-count-badge" aria-label="${count} books">${count}</span></span><h2>${escapeHTML(c.name)}</h2><p>${escapeHTML(detail)}</p></button></article>`
}
function collectionRowHTML(c) {
  const count = Number(c.count) || 0, cover = (c.covers || [])[0] || ''
  const detail = c.rule ? describeRule(c.rule) : ''
  return `<button class="collection-row" data-id="${escapeHTML(c.id)}"><span class="collection-row-cover"><img class="cover" decoding="async" data-src="${escapeHTML(cover)}" data-title="${escapeHTML(c.name)}" alt=""></span><strong>${escapeHTML(c.name)}</strong><span class="collection-row-detail">${escapeHTML(detail)}</span><b class="collection-row-count">${count} ${count === 1 ? 'book' : 'books'}</b></button>`
}
// The address bar follows the open collection, so a refresh comes back to it.
let pendingOpen = ''
let viewRequest = 0
let pendingDetail = ''
let pendingOverview = false

// A collection can hold thousands of books, so its detail is a window like the
// shelf: one page at a time, and the DOM trimmed to the same ceiling. Without
// the injected helpers (tests, older dependency) it degrades to one flat page.
const COLLECTION_PAGE_SIZE = 120
const COLLECTION_WINDOW_MAXIMUM = 600
const COLLECTION_WINDOW_TRIM = 120
const inertWindow = () => ({ reset() {}, begin: () => ({}), finish: () => ({ start: 0, end: 0 }) })
let detail = null

const announcePlace = () => document.defaultView?.dispatchEvent(new Event('bobobooks:place'))
function openOnLoad(id) { pendingOpen = id || '' }
function openId() { return $('#collection-index').hidden ? '' : $('#collection-index').dataset.open || '' }

function closeDetail() {
  if (!detail) return
  detail.pager?.disconnect()
  detail.earlierPager?.disconnect()
  detail = null
}

function collectionTotal(page, fallback) {
  const header = page?.response?.headers?.get?.('X-Total-Count')
  if (header === null || header === undefined || header === '') return fallback
  const total = Number(header)
  return Number.isFinite(total) && total >= 0 ? total : fallback
}

function detailShell(collection, books, total) {
  const layout = getItemLayout()
  return `<div class="series-focus-head"><button class="back-button" aria-label="Back to collections"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 5-7 7 7 7"/></svg></button><div><h2>${escapeHTML(collection.name)}</h2><p class="collection-detail-count">${total} ${total === 1 ? 'book' : 'books'}</p></div></div><div class="collection-detail"><div class="pagination-sentinel collection-earlier-sentinel" aria-hidden="true"></div><button type="button" class="load-more collection-earlier" hidden>Load earlier books</button><div class="virtual-spacer collection-top-spacer" aria-hidden="true"></div><div class="grid book-grid${layout === 'list' ? ' book-list' : ''}">${books.map(book => bookMarkup(book, layout)).join('')}</div><div class="pagination-sentinel collection-sentinel" aria-hidden="true"></div><button type="button" class="load-more collection-more" hidden>Load more books</button><div class="virtual-spacer collection-bottom-spacer" aria-hidden="true"></div></div>`
}

function paintDetailCount() {
  if (!detail) return
  const label = `${detail.total} ${detail.total === 1 ? 'book' : 'books'}`
  $('#status').textContent = label
  const count = detail.root.querySelector?.('.collection-detail-count')
  if (count) count.textContent = label
}

function installDetailPager(state) {
  if (typeof createContinuousPager !== 'function') return
  const sentinel = state.root.querySelector?.('.collection-sentinel')
  const button = state.root.querySelector?.('.collection-more')
  if (!sentinel || !button) return
  state.pager?.disconnect()
  state.pager = createContinuousPager({
    sentinel, button, label: 'books',
    canLoad: () => detail === state && state.startOffset + state.books.length < state.total && isActive() && !state.root.hidden && state.root.dataset.open === state.id,
    load: () => loadMoreCollection(state),
  })
}

// The bounded window drops the oldest cards off the top, so scrolling back up
// used to show a blank spacer where they were. This fetches them again at the
// offset the window now starts at.
function installEarlierPager(state) {
  if (typeof createContinuousPager !== 'function') return
  const sentinel = state.root.querySelector?.('.collection-earlier-sentinel')
  const button = state.root.querySelector?.('.collection-earlier')
  if (!sentinel || !button) return
  state.earlierPager?.disconnect()
  state.earlierPager = createContinuousPager({
    sentinel, button, label: 'earlier books', rootMargin: '400px',
    canLoad: () => detail === state && state.startOffset > 0 && isActive() && !state.root.hidden && state.root.dataset.open === state.id,
    load: () => loadEarlierCollection(state),
  })
}

async function loadEarlierCollection(state) {
  if (state.earlier || state.startOffset <= 0) return false
  const offset = Math.max(0, state.startOffset - COLLECTION_PAGE_SIZE)
  const limit = state.startOffset - offset
  state.earlier = (async () => {
    try {
      const page = await requestJSON(`/api/collections/${encodeURIComponent(state.id)}/books?limit=${limit}&offset=${offset}`)
      if (detail !== state) return false
      const books = page.data || []
      if (!books.length) {
        // Nothing before this point: stop offering it.
        state.startOffset = 0
        return false
      }
      const insertion = state.window.begin('prepend')
      state.grid.insertAdjacentHTML('afterbegin', books.map(book => bookMarkup(book, getItemLayout())).join(''))
      const removed = state.window.finish(insertion)
      state.books.unshift(...books)
      // A prepend trims from the far end, so the retained range still ends where
      // the forward pager expects it to.
      if (removed.end) state.books.splice(-removed.end, removed.end)
      state.startOffset = offset
      paintDetailCount()
      installCoverFallbacks(state.grid)
      installCollectionActions(state.grid)
      return true
    } catch (error) {
      if (detail === state) $('#status').textContent = `Could not load earlier books: ${error.message}`
      return false
    } finally {
      state.earlier = null
      if (detail === state) {
        state.earlierPager?.refresh()
        state.pager?.refresh()
      }
    }
  })()
  return state.earlier
}

// Append the next page below the last card. The bounded window keeps the DOM at
// COLLECTION_WINDOW_MAXIMUM cards by dropping the oldest rows off the top (and
// its spacer absorbs their height, so the reading position does not jump).
async function loadMoreCollection(state) {
  if (state.continuation) return state.continuation
  const offset = state.startOffset + state.books.length
  if (offset >= state.total) return false
  state.continuation = (async () => {
    try {
      const page = await requestJSON(`/api/collections/${encodeURIComponent(state.id)}/books?limit=${COLLECTION_PAGE_SIZE}&offset=${offset}`)
      if (detail !== state) return false
      const books = page.data || []
      if (!books.length) { state.total = offset; paintDetailCount(); return false }
      const insertion = state.window.begin('append')
      state.grid.insertAdjacentHTML('beforeend', books.map(book => bookMarkup(book, getItemLayout())).join(''))
      const removed = state.window.finish(insertion)
      state.books.push(...books)
      if (removed.start) { state.books.splice(0, removed.start); state.startOffset += removed.start }
      if (removed.end) state.books.splice(-removed.end, removed.end)
      state.total = collectionTotal(page, Math.max(state.total, state.startOffset + state.books.length))
      paintDetailCount()
      installCoverFallbacks(state.grid)
      installCollectionActions(state.grid)
      return true
    } catch (error) {
      if (detail === state) $('#status').textContent = `Could not load more books: ${error.message}`
      return false
    } finally {
      state.continuation = null
      if (detail === state) {
        state.pager?.refresh()
        if (state.startOffset > 0 && !state.earlierPager) installEarlierPager(state)
        else state.earlierPager?.refresh()
      }
    }
  })()
  return state.continuation
}

async function loadCollections() {
  const request = ++viewRequest
  pendingDetail = ''
  pendingOverview = true
  syncSaveSearch(false)
  closeDetail()
  releaseCovers($('#collection-index'))
  abortBrowse(); $('#series-index').hidden = true; $('#book-catalog').hidden = true; $('#collection-index').hidden = false; $('#alphabet').hidden = true; $('#status').textContent = 'Opening collections…'
  const library = getSelectedLibrary()
  const isCurrent = () => request === viewRequest && library === getSelectedLibrary() && isActive() && !$('#collection-index').hidden
  try { const data = (await requestJSON(`/api/collections?library=${encodeURIComponent(library)}`)).data; if (!isCurrent()) return; collections = data; $('#status').textContent = `${collections.length} ${collections.length === 1 ? 'collection' : 'collections'}`; $('#collection-index').innerHTML = '<button type="button" class="collection-new-smart">New smart collection</button>' + (collections.length ? (getItemLayout() === 'list' ? `<div class="collection-list"><div class="collection-head" aria-hidden="true"><span></span><span>Collection</span><span>Search</span><span>Books</span></div>${collections.map(collectionRowHTML).join('')}</div>` : `<div class="series-cover-grid collection-grid">${collections.map(collectionCardHTML).join('')}</div>`) : '<p class="empty-state">No collections yet. Save a book or series with the bookmark on its card, or make a smart collection that fills itself.</p>'); delete $('#collection-index').dataset.open; $('.collection-new-smart').onclick = () => { pendingMembers = []; pendingRule = {}; $('#collection-name').value = ''; openCollectionDialog() }; installCoverFallbacks($('#collection-index')); $$('.collection-grid button[data-id], .collection-list button[data-id]').forEach(button => button.onclick = () => openCollection(button.dataset.id)); const wanted = pendingOpen; pendingOpen = ''; if (wanted && collections.some(c => c.id === wanted)) return openCollection(wanted); announcePlace() } catch (error) { if (isCurrent()) $('#status').textContent = `Could not open collections: ${error.message}` } finally { if (request === viewRequest) pendingOverview = false }
}
// mountDetail draws a collection detail from books already in hand. Both the
// first open and a Covers/List switch use it: a layout switch must redraw the
// pages the reader has loaded, because reopening from offset 0 would throw them
// away and drop the reader back to the first screen.
function mountDetail({ root, collection, books, total, startOffset, id }) {
  closeDetail()
  releaseCovers(root)
  root.innerHTML = detailShell(collection, books, total)
  root.dataset.open = id
  const grid = root.querySelector?.('.book-grid')
  const state = { id, root, grid, collection, total, startOffset, books: books.slice(), loading: false, continuation: null, pager: null, window: inertWindow() }
  if (grid && typeof createBoundedWindow === 'function') {
    state.window = createBoundedWindow({
      container: grid,
      topSpacer: root.querySelector('.collection-top-spacer'),
      bottomSpacer: root.querySelector('.collection-bottom-spacer'),
      maximum: COLLECTION_WINDOW_MAXIMUM,
      trim: COLLECTION_WINDOW_TRIM,
      onRemove: nodes => nodes.forEach(node => releaseCovers(node)),
    })
  }
  detail = state
  paintDetailCount()
  if (state.startOffset + state.books.length < state.total) installDetailPager(state)
  if (state.startOffset > 0) installEarlierPager(state)
  announcePlace(); $('.collection-index .back-button').onclick = loadCollections
  installCoverFallbacks(root); installCollectionActions(root)
  return state
}

async function openCollection(id) {
  const request = ++viewRequest
  pendingDetail = id
  pendingOverview = false
  const library = getSelectedLibrary()
  const root = $('#collection-index')
  const isCurrent = () => request === viewRequest && library === getSelectedLibrary() && isActive() && !root.hidden
  const collection = collections.find(c => c.id === id)
  $('#status').textContent = `Opening ${collection?.name || 'collection'}…`
  closeDetail()
  releaseCovers(root)
  try {
    const page = await requestJSON(`/api/collections/${encodeURIComponent(id)}/books?limit=${COLLECTION_PAGE_SIZE}&offset=0`)
    if (!isCurrent()) return
    const books = page.data || []
    const total = collectionTotal(page, books.length)
    mountDetail({ root, collection, books, total, startOffset: 0, id })
  } catch (error) {
    if (isCurrent()) $('#status').textContent = error.message
  } finally { if (request === viewRequest) pendingDetail = '' }
}

function bind() {
  // Author and series suggestions come from the library that is open, with how many books each name has.
  for (const [selector, kind] of [['#rule-author', 'authors'], ['#rule-series', 'series']]) {
    attachSuggestions({
      input: $(selector), document,
      fetchItems: async text => (await requestJSON(`/api/${kind}?library=${encodeURIComponent(getSelectedLibrary())}&q=${encodeURIComponent(text)}&limit=8`)).data.map(({ name, count }) => ({ name, count })),
    })
  }
  $('#create-collection').onclick = async () => {
    const name = $('#collection-name').value.trim()
    if (!name) return
    const button = $('#create-collection'); button.disabled = true
    try { const created = await requestJSON('/api/collections', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(pendingRule ? {name, library_id: getSelectedLibrary(), rule: readRule($('#collection-dialog'))} : {name, library_id: pendingMembers[0].library_id}) }); for (const member of pendingMembers) { const added = await fetch(`/api/collections/${encodeURIComponent(created.data.id)}/members`, { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(member) }); if (!added.ok) throw new Error(await added.text()) } $('#collection-name').value = ''; await refreshCollections(); renderCollectionChoices(); if (pendingRule) { pendingRule = null; $('#collection-dialog').close() } if (isActive()) loadCollections() }
    catch (error) { $('#collection-dialog-status').textContent = error.message }
    finally { button.disabled = false }
  }
  
}

// Several books at once, from the selection bar.
function openFor(members) { pendingMembers = members; pendingRule = null; return openCollectionDialog() }

// The catalogues call this after each render: the button only makes sense for a search that found something.
function syncSaveSearch(hasResults) {
  const button = $('#save-search')
  if (!button) return
  const query = $('#search')?.value.trim()
  button.hidden = !(query && hasResults && getSelectedLibrary())
  button.onclick = () => saveSearch(query)
}

// A smart collection keeps the search itself, so it fills up as matching books are added to the library.
function saveSearch(query) {
  const library = getSelectedLibrary()
  if (!library || !query) return
  pendingMembers = []
  pendingRule = { query }
  $('#collection-name').value = query.slice(0, 80)
  return openCollectionDialog()
}

// Switching Covers/List inside an opened collection redraws that collection instead of going back to the list of collections.
function reflow() {
  if ($('#collection-index').hidden) return false
  // The in-flight overview uses the current layout when it renders.
  if (pendingOverview) return true
  const id = pendingDetail || $('#collection-index').dataset.open
  if (!id) return false
  if (detail && detail.id === id && detail.books.length) {
    mountDetail({ root: detail.root, collection: detail.collection, books: detail.books, total: detail.total, startOffset: detail.startOffset, id })
    return true
  }
  openCollection(id)
  return true
}
return { bind, reflow, openOnLoad, openId, installActions: installCollectionActions, load: loadCollections, openFor, saveSearch, syncSaveSearch }
}
