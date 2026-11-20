import '/assets/vendor/foliate-js/view.js'
import { createReaderPositionSync } from '/assets/reader-position-sync.js?v=5'
import { createReaderLocator } from '/assets/reader-locator.js?v=2'
import { bindReaderNavigation } from '/assets/reader-navigation.js?v=4'
import { fixedLayoutPage, positionFromReaderLocation, readingFractionsEquivalent, selectInitialReadingPosition } from '/assets/reader-remote-position.js?v=3'
import { createReadAloud } from '/assets/read-aloud.js?v=3'
import { createRemoteZipFile } from '/assets/remote-zip-file.js?v=1'

import { createReaderJump } from '/assets/reader-jump.js?v=1'
import { mountClassicSearch } from '/assets/reader-search.js?v=3'
import { createReaderSyncOffers } from '/assets/reader-sync-offers.js?v=4'
import { createReaderPositionHistory } from '/assets/reader-position-history.js?v=1'

const largePublicationBytes = 48 * 1024 * 1024

const $ = selector => document.querySelector(selector)
const parts = location.pathname.split('/').filter(Boolean)
const library = parts[1]
const id = parts[2]
const view = $('#reader')
const loading = $('#loading')

async function downloadPublicationBlob(response) {
  if (!response.body?.getReader) return response.blob()
  const total = Number(response.headers.get('Content-Length')) || 0
  const reader = response.body.getReader(), chunks = []
  let received = 0
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    chunks.push(value); received += value.byteLength
    const amount = `${(received / 1048576).toFixed(received < 10485760 ? 1 : 0)} MiB`
    $('#loading-detail').textContent = total > 0
      ? `Downloading from your shelf · ${Math.min(100, Math.round(received / total * 100))}%`
      : `Downloading from your shelf · ${amount}`
  }
  $('#loading-detail').textContent = 'Preparing pages…'
  return new Blob(chunks, { type: response.headers.get('Content-Type') || '' })
}
const savedTheme = localStorage.getItem('mochil-theme') || localStorage.getItem('opdshelf-theme')
const theme = savedTheme || (matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
document.documentElement.dataset.theme = theme
const preferenceKey = 'mochil-reader-settings'
const directionKey = `mochil-reader-direction:${library}:${id}`
const positionKey = `mochil-position:${library}:${id}`
const deviceKey = 'mochil-device-id'
const randomDeviceID = () => crypto.randomUUID?.() || Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')
const deviceID = localStorage.getItem(deviceKey) || randomDeviceID()
const deviceName = navigator.userAgentData?.platform || navigator.platform || 'Web reader'
localStorage.setItem(deviceKey, deviceID)
let savedPreferences = {}
try { savedPreferences = JSON.parse(localStorage.getItem(preferenceKey) || '{}') } catch { localStorage.removeItem(preferenceKey) }
// Direction belongs to this publication. A manga set to RTL must not turn
// unrelated novels and comics RTL in the same browser.
delete savedPreferences.direction
const preferences = { fontSize: 100, lineHeight: 1.6, pageWidth: 720, margin: 24, flow: 'paginated', spread: 1, ...savedPreferences, direction: localStorage.getItem(directionKey) || 'auto' }
const positionStore = {
  load() { try { return JSON.parse(localStorage.getItem(positionKey) || 'null') } catch { return null } },
  save(position, extra = {}) { localStorage.setItem(positionKey, JSON.stringify({ ...position, updatedAt: Math.floor(Date.now() / 1000), deviceID, pending: true, ...extra })) },
  markConflicted() { const current = this.load(); if (current) try { localStorage.setItem(positionKey, JSON.stringify({ ...current, conflicted: true })) } catch {} },
  markSynced(position) {
    const current = this.load()
    if (current?.cfi === position?.cfi && readingFractionsEquivalent(current?.fraction, position?.fraction)) {
      localStorage.setItem(positionKey, JSON.stringify({ ...current, pending: false }))
    }
  },
}
let publicationKind = ''
let publicationFormat = ''
let publicationPageCount = 0

const currentReaderPosition = () => positionFromReaderLocation(view, view.lastLocation, publicationFormat === 'cbz' ? 'comic' : '', publicationPageCount)

const renderLocation = syncState => {
  const fraction = readingSync.latestPosition?.fraction ?? view.lastLocation?.fraction
  const percentage = Number.isFinite(fraction) ? `${Math.round(fraction * 100)}%` : ''
  $('#location').textContent = [percentage, syncState].filter(Boolean).join(' · ')
}
const readingSync = createReaderPositionSync({
  endpoint: `/api/reading/${encodeURIComponent(library)}/${encodeURIComponent(id)}`,
  deviceID,
  deviceName,
  positionStore,
  renderStatus: renderLocation,
})
readingSync.start()
const loadReadingPosition = () => readingSync.load()
const saveReadingPosition = (position, keepalive) => readingSync.save(position, keepalive)
const scheduleReadingSave = position => readingSync.schedule(position)
const { webAnchorFromLocation, resolveTextAnchor, resolveKoReaderPosition } = createReaderLocator(view)

const { goToServerPosition, checkForRemotePosition, offerKoSyncPosition } = createReaderSyncOffers({
  $, view, library, id, deviceID, readingSync, resolveTextAnchor, resolveKoReaderPosition, currentReaderPosition,
  fixedLayoutKind: () => (publicationFormat === 'cbz' ? 'comic' : ''),
  pageCount: () => publicationPageCount,
})

const positionHistory = createReaderPositionHistory({
  $, endpoint: `/api/reading/${encodeURIComponent(library)}/${encodeURIComponent(id)}`,
  deviceID, readingSync, goToPosition: goToServerPosition,
  currentPosition: currentReaderPosition, renderLocation,
})

function showProgress(fraction) {
  if (!Number.isFinite(fraction)) return
  const value = Math.min(1, Math.max(0, fraction))
  document.documentElement.style.setProperty('--progress', String(value))
  const seek = $('#seek')
  if (seek && document.activeElement !== seek) seek.value = String(Math.round(value * 1000))
}

function bookStyles() {
  const ink = theme === 'dark' ? '#f4eee6' : '#241f1a'
  const paper = theme === 'dark' ? '#171513' : '#f5efe6'
  const readingDirection = resolvedReadingDirection()
  const direction = readingDirection === 'auto' ? '' : `direction:${readingDirection}!important;`
  return `:root{color-scheme:${theme};background:${paper};color:${ink};font-size:${preferences.fontSize}%!important;${direction}} body{line-height:${preferences.lineHeight};${direction}} p,li,blockquote,dd{line-height:${preferences.lineHeight}!important;hyphens:auto;-webkit-hyphens:auto;overflow-wrap:break-word;text-wrap:pretty} .-epub-media-overlay-active,.-epub-media-overlay-active *{background-color:${theme === 'dark' ? 'rgba(240,154,114,.30)' : 'rgba(200,93,58,.22)'}!important;color:inherit!important;border-radius:3px;box-decoration-break:clone}`
}

function resolvedReadingDirection() {
  return preferences.direction === 'auto' && publicationKind === 'manga' ? 'rtl' : preferences.direction
}

function applyPreferences() {
  const readingDirection = resolvedReadingDirection()
  if (view.isFixedLayout && readingDirection !== 'auto' && view.book?.dir !== readingDirection) {
    const lastLocation = view.lastLocation?.cfi
    view.book.dir = readingDirection
    view.renderer.open(view.book)
    if (lastLocation) view.goTo(lastLocation)
  }
  view.renderer?.setAttribute('flow', preferences.flow)
  view.renderer?.setAttribute('margin', `${preferences.margin}px`)
  view.renderer?.setAttribute('max-inline-size', `${preferences.pageWidth}px`)
  view.renderer?.setAttribute('max-column-count', String(preferences.spread))
  view.renderer?.setStyles?.(bookStyles())
  view.renderer?.render?.()
  const { direction, ...globalPreferences } = preferences
  localStorage.setItem(preferenceKey, JSON.stringify(globalPreferences))
  localStorage.setItem(directionKey, direction)
}

const bookURL = `/read/${encodeURIComponent(library)}/${encodeURIComponent(id)}`
try {
  const response = await fetch(`/api/books/${encodeURIComponent(library)}/${encodeURIComponent(id)}`)
  if (!response.ok) throw new Error('Book metadata is unavailable')
  const meta = await response.json()
  publicationKind = meta.library_kind || ''
  publicationFormat = String(meta.format || '').toLowerCase()
  publicationPageCount = Math.trunc(Number(meta.page_count)) || 0
  $('#title').textContent = meta.title
  document.title = `${meta.title} · Mochil`
  $('#loading-detail').textContent = 'Downloading from your shelf…'

  if (meta.page_stream_url && String(meta.format).toLowerCase() === 'cbz') {
    $('#loading-detail').textContent = 'Preparing streamed pages…'
    const { makeStreamedComicBook } = await import('/assets/streamed-comic-book.js?v=1')
    await view.open(makeStreamedComicBook({ title: meta.title, pageCount: meta.page_count, pageURL: meta.page_stream_url }))
  } else {
    const publicationName = meta.path?.split('/').pop() || `${meta.title}.${meta.format || 'epub'}`
    const lastModified = meta.modified ? Date.parse(meta.modified) : Date.now()
    if (publicationFormat === 'epub' && Number(meta.size) >= largePublicationBytes) {
      // Read-aloud EPUBs run to hundreds of megabytes: read them by byte range.
      $('#loading-detail').textContent = 'Opening your book…'
      await view.open(createRemoteZipFile({ url: bookURL, name: publicationName, type: meta.media_type, size: Number(meta.size), lastModified }))
    } else {
      const publicationResponse = await fetch(bookURL)
      if (!publicationResponse.ok) throw new Error(`Download returned HTTP ${publicationResponse.status}`)
      const publicationBlob = await downloadPublicationBlob(publicationResponse)
      const publicationFile = new File([publicationBlob], publicationName, {
        type: meta.media_type || publicationBlob.type,
        lastModified,
      })
      await view.open(publicationFile)
    }
  }
  const initialDirection = resolvedReadingDirection()
  if (view.isFixedLayout && initialDirection !== 'auto') {
    view.book.dir = initialDirection
    view.renderer.open(view.book)
  }
  applyPreferences()
  createReadAloud({
    view, button: $('#listen'), speedField: $('#listen-speed-field'), speedSelect: $('#listen-speed'),
    sleepField: $('#listen-sleep-field'), sleepSelect: $('#listen-sleep'),
    storage: localStorage, mediaSession: navigator.mediaSession, getTitle: () => meta.title,
    onError: () => { $('#location').textContent = 'Audio is unavailable for this part' },
    onSleep: () => { $('#location').textContent = 'Sleep timer ended' },
  })
  const serverPosition = await loadReadingPosition().catch(() => null)
  const localPosition = positionStore.load()
  const initialPosition = selectInitialReadingPosition(localPosition, serverPosition)
  const savedPosition = initialPosition.position
  // Foliate resolves a position object by fraction before looking at its CFI.
  // Pass the CFI itself so reopening uses the exact visible text anchor instead
  // of an approximate whole-book percentage that can round to the next page.
  const exactPage = initialPosition.source === 'remote'
    ? fixedLayoutPage(savedPosition, publicationFormat === 'cbz' ? 'comic' : '', publicationPageCount)
    : null
  // Another reader (Android, KOReader, the Readium preview) saves a text anchor
  // instead of an EPUB CFI; find that text so this reader opens on the same page.
  const anchoredCfi = !exactPage && !savedPosition?.cfi && initialPosition.source === 'remote' && savedPosition?.web_anchor
    ? (await resolveTextAnchor(savedPosition.web_anchor).catch(() => null))?.cfi
    : null
  const restoreTarget = (exactPage ?? (savedPosition?.cfi || anchoredCfi)) || (Number.isFinite(savedPosition?.fraction)
    ? { fraction: savedPosition.fraction }
    : null)
  await view.init({ lastLocation: restoreTarget, showTextStart: false })
  $('#location').textContent = view.lastLocation?.fraction != null ? `${Math.round(view.lastLocation.fraction * 100)}%` : ''
  showProgress(view.lastLocation?.fraction)
  loading.remove()
  if (initialPosition.source === 'local') {
    const position = currentReaderPosition()
    if (position) scheduleReadingSave(position)
  } else {
    const position = currentReaderPosition()
    if (position) readingSync.acceptRemote(position, 'Synced')
  }

  view.addEventListener('relocate', event => {
    const detail = event.detail || {}
	readingSync.latestAnchor = webAnchorFromLocation(detail)
    $('#location').textContent = detail.fraction != null ? `${Math.round(detail.fraction * 100)}%` : ''
    showProgress(detail.fraction)
    if (view.lastLocation?.fraction != null) {
      const position = currentReaderPosition()
      if (!position) return
      if (readingSync.applyingRemote) {
		readingSync.acceptRemote(position)
	  } else {
        positionStore.save(position)
        scheduleReadingSave(position)
      }
    }
  })
  offerKoSyncPosition(serverPosition, { auto: true }).catch(() => { readingSync.applyingRemote = false })
  const remotePositionPoll = setInterval(() => checkForRemotePosition().catch(() => {}), 30000)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') checkForRemotePosition().catch(() => {})
  })
  window.addEventListener('pagehide', () => clearInterval(remotePositionPoll))
  bindReaderNavigation({
    document, view, previousButton: $('#prev'), nextButton: $('#next'), onActivity: () => wake(),
    controlsVisible: () => document.body.classList.contains('controls-visible'),
    hideControls: () => { clearTimeout(timer); document.body.classList.remove('controls-visible') },
    isScrolled: () => view.renderer?.getAttribute?.('flow') === 'scrolled',
  })

  const dialog = $('#contents')
  const list = $('#toc-list')
  const toc = view.book?.toc || []
  const jumpTo = createReaderJump({ view, readingSync, currentReaderPosition, positionStore, saveReadingPosition })
  for (const item of toc) {
    const button = document.createElement('button')
    button.textContent = item.label
    button.onclick = async () => {
      button.disabled = true
      try {
        await jumpTo(item.href)
        dialog.close()
      } finally {
        button.disabled = false
      }
    }
    list.append(button)
  }
  if (!toc.length) list.textContent = 'This book does not provide a table of contents.'
  if (publicationFormat === 'epub') {
    $('#search-open').hidden = false
    mountClassicSearch({ $, view, jumpTo })
  }
  $('#toc').onclick = () => dialog.showModal()
  $('#close-toc').onclick = () => dialog.close()
  $('#fullscreen').onclick = () => document.fullscreenElement ? document.exitFullscreen() : document.documentElement.requestFullscreen()
  const options = $('#reading-options')
  const fields = {
    fontSize: $('#font-size'), lineHeight: $('#line-height'), pageWidth: $('#page-width'),
    margin: $('#page-margin'), flow: $('#reader-flow'), spread: $('#reader-spread'), direction: $('#reader-direction'),
  }
  if (publicationKind === 'manga') fields.direction.options[0].textContent = 'Manga default (right to left)'
  for (const [key, field] of Object.entries(fields)) field.value = preferences[key]
  const updateOutputs = () => { $('#font-size-output').value = `${preferences.fontSize}%`; $('#line-height-output').value = Number(preferences.lineHeight).toFixed(1) }
  updateOutputs()
  for (const [key, field] of Object.entries(fields)) field.oninput = () => {
    preferences[key] = ['fontSize', 'lineHeight', 'pageWidth', 'margin', 'spread'].includes(key) ? Number(field.value) : field.value
    updateOutputs()
    applyPreferences()
  }
  $('#reader-settings').onclick = () => options.showModal()
  $('#close-settings').onclick = () => options.close()
  $('#sync-now').onclick = async () => {
    const button = $('#sync-now')
    button.disabled = true
    const original = button.textContent
    button.textContent = 'Checking…'
    try {
      options.close()
      const offered = await checkForRemotePosition()
      if (!offered) button.textContent = 'Already up to date'
    } catch {
      button.textContent = 'Could not check'
    }
    setTimeout(() => { button.textContent = original; button.disabled = false }, 1800)
  }
  $('#reading-history').onclick = () => {
    options.close()
    positionHistory.open()
  }
  $('#close-history').onclick = () => $('#position-history').close()
  $('#restore-cancel').onclick = () => $('#restore-position').close()
  $('#mark-finished').onclick = async () => {
    const response = await fetch(`/api/reading/${encodeURIComponent(library)}/${encodeURIComponent(id)}/finish`, { method: 'POST' })
    if (response.ok) location.href = '/'
  }

  const seek = $('#seek')
  seek.addEventListener('input', () => { $('#location').textContent = `${Math.round(Number(seek.value) / 10)}%` })
  seek.addEventListener('change', () => { view.goToFraction(Number(seek.value) / 1000).catch(() => {}) })

  var timer
  function wake() {
    document.body.classList.add('controls-visible')
    clearTimeout(timer)
    timer = setTimeout(() => document.body.classList.remove('controls-visible'), 2400)
  }
  for (const event of ['pointermove', 'pointerdown', 'keydown']) document.addEventListener(event, wake)
  wake()
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && !dialog.open && !options.open) leaveReader(event)
  })

  let leaving = false
  const leaveReader = async event => {
    event?.preventDefault()
    if (leaving) return
    leaving = true
    readingSync.cancelScheduled()
    const position = readingSync.latestPosition || currentReaderPosition()
    if (position) await saveReadingPosition(position).catch(() => {})
    location.href = '/'
  }
  $('#reader-back').addEventListener('click', leaveReader)
  document.addEventListener('visibilitychange', () => {
    const position = readingSync.latestPosition || currentReaderPosition()
    if (document.visibilityState === 'hidden' && position) {
      readingSync.cancelScheduled()
      saveReadingPosition(position, true).catch(() => {})
    }
  })
} catch (error) {
  console.error('Reader failed to open', error)
  loading.querySelector('strong').textContent = 'This book could not be opened'
  const detail = $('#loading-detail') || loading.appendChild(document.createElement('p'))
  detail.textContent = `${error.message}. Return to the library and try again.`
}
