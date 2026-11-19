import { watchAudioTable } from './audio-list-table.js?v=2'

export function createAudioSurfaceControls({ $, musicDestinations, audiobookDestinations }) {
  const podcastOnly = [
    $('#podcast-subscriptions'),
    $('#podcast-manage-toggle'),
    $('#podcast-layout-field'),
    $('#podcast-state-field'),
  ]
  const audiobookOnly = [
    $('#audiobook-series-layout-field'),
    $('#audiobook-item-layout-field'),
  ]
  const musicLayout = $('#music-layout-field')
  const playlistCreate = $('#playlist-create')
  const seriesSort = $('#audio-sort option[value="series"]')
  $('#music-browse-field').hidden = true
  const toggle = $('#audio-layout-toggle')
  watchAudioTable($('#audio-items'))
  let kind = ''

  // Cover/list sits in the toolbar like the other shelves; it drives whichever layout setting fits what is on screen.
  const layoutSelect = () => {
    // Discover is a curated overview whose sections lay themselves out, so it has no cover/list choice.
    if (kind === 'music') return $('#music-browse').value === 'discover' ? null : $('#music-layout')
    if (kind === 'podcasts') return $('#podcast-layout')
    if (kind === 'audiobooks') return ['series', 'authors'].includes($('#audiobook-browse').value) ? $('#audiobook-series-layout') : $('#audiobook-item-layout')
    return null
  }
  const sync = () => {
    const select = layoutSelect()
    toggle.hidden = !select
    toggle.querySelectorAll('[data-audio-layout]').forEach(button => button.setAttribute('aria-pressed', String(!!select && button.dataset.audioLayout === select.value)))
  }
  toggle.addEventListener('click', event => {
    const button = event.target.closest('[data-audio-layout]'), select = layoutSelect()
    if (!button || !select || select.value === button.dataset.audioLayout) return
    select.value = button.dataset.audioLayout
    select.dispatchEvent(new Event('change', { bubbles: true }))
    sync()
  })
  toggle.ownerDocument.addEventListener('click', () => setTimeout(sync, 0))

  return {
    show(nextKind) {
      kind = nextKind
      const podcasts = kind === 'podcasts'
      const audiobooks = kind === 'audiobooks'
      const music = kind === 'music'
      podcastOnly.forEach(element => { element.hidden = !podcasts })
      audiobookOnly.forEach(element => { element.hidden = !audiobooks })
      seriesSort.hidden = !audiobooks
      musicLayout.hidden = !music
      musicDestinations.setVisible(music)
      audiobookDestinations.setVisible(audiobooks)
      playlistCreate.hidden = true
      sync()
    },
  }
}
