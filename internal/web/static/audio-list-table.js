const HEAD = '<div class="audio-table-head" aria-hidden="true"><span></span><span>Title</span><span>Artist</span><span>Album</span><span>Time</span></div>'

/** A header row above music lists that show their columns (wide screens only; the stylesheet hides it elsewhere). */
export function watchAudioTable(list) {
  if (!list || typeof MutationObserver !== 'function') return
  let queued = false
  function sync() {
    queued = false
    const wants = Boolean(list.querySelector(':scope > .audio-item .audio-col'))
    const head = list.previousElementSibling?.classList.contains('audio-table-head') ? list.previousElementSibling : null
    if (wants && !head) list.insertAdjacentHTML('beforebegin', HEAD)
    else if (!wants && head) head.remove()
  }
  new MutationObserver(() => { if (!queued) { queued = true; queueMicrotask(sync) } }).observe(list, { childList: true })
}
