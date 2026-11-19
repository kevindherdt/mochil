const PAGE_SIZE = 60

export function queuePage(items, active, key, requestedOffset = 0, followActive = true) {
  const length = items.length
  const activeIndex = active ? items.findIndex(item => key(item) === key(active)) : -1
  const centered = !followActive || activeIndex < 0 ? requestedOffset : activeIndex - Math.floor(PAGE_SIZE / 2)
  const offset = Math.max(0, Math.min(centered, Math.max(0, length - PAGE_SIZE)))
  return { items: items.slice(offset, offset + PAGE_SIZE), offset, activeIndex, total: length }
}

/** Owns the bounded queue presented inside expanded Now Playing. */
export function createAudioPlayerQueue({ $, escapeHTML, key, onSelect, onShow = () => {}, getSyncError = () => '', onRetry = () => {} }) {
  const panel = $('#audio-player-queue')
  const toggle = $('#audio-queue-toggle')
  let queue = []
  let active
  let requestedOffset = 0
  let followActive = true
  let renderedPage

  function render() {
    const list = $('#audio-queue-items')
    const focused = list.ownerDocument?.activeElement
    const retryFocused = focused?.hasAttribute?.('data-queue-retry') && list.contains(focused)
    const focusedIndex = list.contains(focused) ? Number(focused.dataset.queueIndex) : -1
    const focusedItem = renderedPage?.items[focusedIndex - renderedPage.offset]
    const page = queuePage(queue, active, key, requestedOffset, followActive)
    requestedOffset = page.offset
    $('#audio-queue-position').textContent = page.total
      ? `${Math.max(1, page.activeIndex + 1)} of ${page.total.toLocaleString()}`
      : 'Empty'
    const error = getSyncError()
    const notice = error ? `<p role="status">${escapeHTML(error)}</p><button data-queue-retry>Retry queue sync</button>` : ''
    list.innerHTML = notice + (page.items.length
      ? page.items.map((item, index) => {
          const current = active && key(item) === key(active)
          const position = page.offset + index
          const context = [item.artist, item.album].filter(Boolean).join(' · ')
          return `<button data-queue-index="${position}"${current ? ' class="active" aria-current="true"' : ''}><span><strong>${escapeHTML(item.title || 'Untitled')}</strong><small>${escapeHTML(context || 'Unknown artist')}</small></span><b>${position + 1}</b></button>`
        }).join('')
      : error ? '' : '<p>No tracks in the current queue.</p>')
    $('#audio-queue-earlier').disabled = page.offset === 0
    $('#audio-queue-later').disabled = page.offset + page.items.length >= page.total
    if (focusedItem) {
      const match = page.items.findIndex(item => key(item) === key(focusedItem))
      const index = match >= 0 ? page.offset + match
        : Math.max(page.offset, Math.min(focusedIndex, page.offset + page.items.length - 1))
      const target = list.querySelector(`[data-queue-index="${index}"]`) || $('#audio-queue-close')
      target.focus({ preventScroll: true })
    }
    if (retryFocused) (list.querySelector('[data-queue-retry]') || $('#audio-queue-close')).focus({preventScroll:true})
    renderedPage = page
  }

  function close({ restoreFocus = false } = {}) {
    if (panel.hidden) return
    panel.hidden = true
    toggle.setAttribute('aria-expanded', 'false')
    if (restoreFocus) toggle.focus()
  }

  function show({ focus = true } = {}) {
    onShow()
    followActive = true
    render()
    panel.hidden = false
    toggle.setAttribute('aria-expanded', 'true')
    if (!focus) return
    const target = panel.querySelector('[aria-current="true"]')
      || panel.querySelector('[data-queue-index]') || $('#audio-queue-close')
    target.focus()
  }

  toggle.onclick = () => panel.hidden ? show() : close({ restoreFocus: true })
  $('#audio-queue-close').onclick = () => close({ restoreFocus: true })
  $('#audio-queue-earlier').onclick = () => { requestedOffset -= PAGE_SIZE; followActive = false; render() }
  $('#audio-queue-later').onclick = () => { requestedOffset += PAGE_SIZE; followActive = false; render() }
  panel.onkeydown = event => {
    if (event.key === 'Escape') { event.preventDefault(); close({ restoreFocus: true }) }
  }
  $('#audio-queue-items').onclick = event => {
    if (event.target.closest('[data-queue-retry]')) { onRetry(); render(); return }
    const button = event.target.closest('[data-queue-index]')
    const item = queue[Number(button?.dataset.queueIndex)]
    if (item) onSelect(item)
  }

  function sync(items, current) {
    if ((current ? key(current) : '') !== (active ? key(active) : '')) followActive = true
    queue = Array.isArray(items) ? items : []
    active = current
    toggle.hidden = current?.library_kind !== 'music'
    if (!panel.hidden) render()
    if (toggle.hidden) close()
    dock()
  }

  // On a big screen the expanded player keeps the queue open beside the cover, like a desktop player.
  const player = panel.closest?.('.audio-player'), wide = globalThis.matchMedia?.('(min-width:1100px) and (min-height:620px)')
  let docked = false
  function dock() {
    if (!player || !wide) return
    const want = player.classList.contains('expanded') && wide.matches && !toggle.hidden
    if (want && panel.hidden) show({ focus: false })
    else if (!want && docked) close()
    docked = want
  }
  if (player && wide && typeof MutationObserver === 'function') {
    new MutationObserver(dock).observe(player, { attributes: true, attributeFilter: ['class'] })
    wide.addEventListener?.('change', dock)
  }

  return { close, sync }
}

/** Connect queue synchronization without expanding the library coordinator. */
export function createMusicPlayerQueue({music, open, ...options}) {
  return createAudioPlayerQueue({
    ...options,
    onSelect:item=>open(item,true),
    getSyncError:()=>music.loadErrors.queue || '',
    onRetry:()=>music.retryQueueSave(),
  })
}
