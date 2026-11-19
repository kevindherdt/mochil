export function createAudiobookChapters({ $, audio, getAudio = () => audio, claimSeek = () => {}, player, getActive, clock, escapeHTML, expandPlayer, updateTimeline }) {
  let chapterIndex = -1

  function revealCurrent() {
    const list = $('#audio-chapter-list'), selected = list.querySelector('.active')
    if (!selected || $('#audio-chapters').hidden) return
    const row = selected.getBoundingClientRect(), viewport = list.getBoundingClientRect()
    if (row.top < viewport.top) list.scrollTop += row.top - viewport.top
    else if (row.bottom > viewport.bottom) list.scrollTop += row.bottom - viewport.bottom
  }

  function current() {
    const position = getAudio().currentTime
    return getActive()?.chapters?.find(chapter => position >= chapter.start_seconds && position < chapter.end_seconds)
  }
  function update(chapter = current()) {
    const active = getActive(), next = active?.chapters?.indexOf(chapter) ?? -1
    if (next === chapterIndex) return chapter
    chapterIndex = next
    $('#audio-chapter-list').querySelectorAll('[data-chapter]').forEach((button, index) => {
      const selected = index === next
      button.classList.toggle('active', selected)
      button.setAttribute('aria-current', selected ? 'true' : 'false')
    })
    revealCurrent()
    return chapter
  }
  function openItem(item) {
    const chapters = item?.chapters || []
    close()
    $('#audio-chapters-toggle').hidden = item?.library_kind !== 'audiobooks' || !chapters.length
    $('#audio-chapter-list').innerHTML = chapters.map((chapter, index) => `<button data-chapter="${index}"><span><strong>${escapeHTML(chapter.title || `Chapter ${index + 1}`)}</strong><small>${clock(chapter.start_seconds)}</small></span></button>`).join('')
    chapterIndex = -1
    update()
  }
  function close() {
    const panel = $('#audio-chapters')
    const restore = panel.contains(player.ownerDocument.activeElement)
    panel.hidden = true
    $('#audio-chapters-toggle').setAttribute('aria-expanded', 'false')
    if (restore && !player.hidden) player.querySelector('.audio-player-more summary')?.focus({preventScroll:true})
  }

  $('#audio-chapters-toggle').onclick = () => {
    expandPlayer()
    $('#audio-chapters').hidden = false
    $('#audio-chapters-toggle').setAttribute('aria-expanded', 'true')
    player.querySelector('.audio-player-more').open = false
    update()
    revealCurrent()
    const target = $('#audio-chapter-list').querySelector('.active') || $('#audio-chapters-close')
    target.focus({preventScroll:true})
  }
  $('#audio-chapters').addEventListener('keydown', event => {
    if (event.key !== 'Escape' || $('#audio-chapters').hidden) return
    event.preventDefault()
    event.stopPropagation()
    close()
  })
  $('#audio-chapters-close').onclick = close
  $('#audio-chapter-list').onclick = event => {
    const button = event.target.closest('[data-chapter]'), chapter = getActive()?.chapters?.[Number(button?.dataset.chapter)]
    if (!chapter) return
    const audio = getAudio()
    claimSeek()
    audio.currentTime = chapter.start_seconds
    close()
    updateTimeline()
    audio.play().catch(() => {})
  }

  return { current, update, openItem, close }
}
