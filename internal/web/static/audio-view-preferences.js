/** Owns persisted audio browse controls; catalogue reload policy stays with the coordinator. */
export function createAudioViewPreferences({
  storage = localStorage,
  viewMenu,
  sort,
  podcastState,
  podcastLayout,
  musicLayout,
  audiobookSeriesLayout,
  audiobookItemLayout,
  activeKind,
  onSort,
  onPodcastState,
  onPodcastLayout,
  onMusicLayout,
  onAudiobookSeriesLayout,
  onAudiobookItemLayout,
}) {
  const closeMenu = () => { if (viewMenu) viewMenu.open = false }
  const bind = (element, key, fallback, onChange) => {
    element.value = storage.getItem(key) || fallback
    element.onchange = () => {
      closeMenu()
      storage.setItem(key, element.value)
      onChange()
    }
  }

  const savedSort = kind => storage.getItem(`mochil-audio-sort-${kind}`)
  sort.onchange = () => {
    closeMenu()
    storage.setItem(`mochil-audio-sort-${activeKind()}`, sort.value)
    onSort()
  }
  bind(podcastState, 'mochil-podcast-state', 'all', onPodcastState)
  bind(podcastLayout, 'mochil-podcast-layout', 'covers', onPodcastLayout)
  bind(musicLayout, 'mochil-music-layout', 'covers', onMusicLayout)
  bind(audiobookSeriesLayout, 'mochil-audiobook-series-layout', 'covers', onAudiobookSeriesLayout)
  bind(audiobookItemLayout, 'mochil-audiobook-item-layout', 'covers', onAudiobookItemLayout)

  return { closeMenu, savedSort }
}
