/** Owns delegated catalogue clicks; state and rendering remain with the coordinator. */
export function bindAudioLibraryInteractions({
  container, playlistController, musicCatalogue, podcastCatalogue,
  getItems, findItem, groupLabel, getActive, getKind, getBrowse, getDiscovery,
  openMusicGroup, closeMusicGroup, loadPodcastShow, loadAudiobookGroup,
  loadAudiobookRoot, markPodcastPlayed, open, setQueue, setShuffle,
  orderedQueue, playNext, addToQueue, toggleFavorite, openTrackInfo,
  clearQueue, retryQueueSave, removeFromQueue, moveInQueue, render, confirmClear,
  selectBrowse, selectSort, selectPlaylist, reload, findDiscoveryGroup,
}) {
  let listScroll = 0
  container.onclick = async event => {
    if (await playlistController.handleClick(event)) return
    const musicGroup = event.target.closest('[data-music-group]')
    if (musicGroup) {
      const group = musicCatalogue.findGroup(musicGroup.dataset.musicGroup)
      if (group) await openMusicGroup(group)
      return
    }
    if (event.target.closest('[data-music-group-back]')) { await closeMusicGroup(); return }
    const groupPlayback = event.target.closest('[data-play-music-group],[data-shuffle-music-group]')
    if (groupPlayback) {
      const items = getItems()
      if (items.length) {
        setQueue(items)
        setShuffle(groupPlayback.hasAttribute('data-shuffle-music-group'))
        open(orderedQueue()[0], true)
      }
      return
    }
    const podcast = event.target.closest('[data-podcast]')
    if (podcast) { listScroll = window.scrollY; await loadPodcastShow(podcast.dataset.podcast); return }
    if (event.target.closest('.podcast-back')) { podcastCatalogue.closeShow(); render(); window.scrollTo(0, listScroll); return }
    const audiobookSeries = event.target.closest('[data-audiobook-series]')
    if (audiobookSeries) { listScroll = window.scrollY; await loadAudiobookGroup(audiobookSeries.dataset.audiobookSeries); return }
    if (event.target.closest('.audiobook-back')) { await loadAudiobookRoot(); window.scrollTo(0, listScroll); return }
    const group = event.target.closest('[data-play-group]')
    if (group) {
      const list = getItems().filter(item => groupLabel(item) === group.dataset.playGroup)
      setQueue(list)
      open(list[0], true)
      return
    }
    const item = findItem(event.target)
    if (!item) return
    if (event.target.closest('.audio-open')) open(item)
    else if (event.target.closest('[data-action="played"]')) await markPodcastPlayed(item, !item.progress?.finished)
    else if (event.target.closest('[data-action="next"]')) await playNext(item, getActive())
    else if (event.target.closest('[data-action="queue"]')) await addToQueue(item)
    else if (event.target.closest('[data-action="favorite"]')) await toggleFavorite(item)
    else if (event.target.closest('[data-action="track-info"]')) openTrackInfo(item)
    else await playlistController.handleItemAction(event, item)
  }

  container.addEventListener('click', event => {
    if (getKind() !== 'music' || getBrowse() !== 'queue') return
    if (event.target.closest('[data-retry-queue-save]')) {
      event.stopImmediatePropagation()
      retryQueueSave()
      render()
      return
    }
    const action = event.target.closest('[data-action]')?.dataset.action
    const item = findItem(event.target)
    if (event.target.closest('[data-clear-queue]')) {
      event.stopImmediatePropagation()
      if (confirmClear()) { clearQueue(); render() }
      return
    }
    if (event.target.closest('.audio-open') && item) {
      event.stopImmediatePropagation()
      open(item, true)
      return
    }
    if (!item || !['queue-up', 'queue-down', 'queue-remove'].includes(action)) return
    event.stopImmediatePropagation()
    if (action === 'queue-remove') removeFromQueue(item)
    else moveInQueue(item, action === 'queue-up' ? -1 : 1)
    render()
  }, true)

  container.addEventListener('click', async event => {
    const destinationElement = event.target.closest('[data-discovery-destination]')
    const destination = destinationElement?.dataset.discoveryDestination
    const group = event.target.closest('[data-discovery-group]')
    const playlist = event.target.closest('[data-discovery-playlist]')
    if (!destination && !group && !playlist) return
    event.stopImmediatePropagation()
    if (playlist) {
      selectBrowse('playlists')
      selectPlaylist(playlist.dataset.discoveryPlaylist)
      render()
      return
    }
    const browse = destination || group.dataset.discoveryType
    const selected = group ? findDiscoveryGroup(getDiscovery(), browse, group.dataset.discoveryGroup) : null
    selectBrowse(browse)
    if (destinationElement?.dataset.discoverySort) selectSort(destinationElement.dataset.discoverySort)
    await reload()
    if (selected) await openMusicGroup(selected)
  }, true)
}

export function discoveryGroup(discovery, browse, name) {
  return discovery?.[browse]?.find(value => value.name === name) || null
}
