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

  const savedSort = kind => storage.getItem(`bobobooks-audio-sort-${kind}`)
  sort.onchange = () => {
    closeMenu()
    storage.setItem(`bobobooks-audio-sort-${activeKind()}`, sort.value)
    onSort()
  }
  bind(podcastState, 'bobobooks-podcast-state', 'all', onPodcastState)
  bind(podcastLayout, 'bobobooks-podcast-layout', 'covers', onPodcastLayout)
  bind(musicLayout, 'bobobooks-music-layout', 'covers', onMusicLayout)
  bind(audiobookSeriesLayout, 'bobobooks-audiobook-series-layout', 'covers', onAudiobookSeriesLayout)
  bind(audiobookItemLayout, 'bobobooks-audiobook-item-layout', 'covers', onAudiobookItemLayout)

  return { closeMenu, savedSort }
}
