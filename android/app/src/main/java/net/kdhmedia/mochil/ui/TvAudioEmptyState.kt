package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicState

/** Match TvAudioGrid's branch without counting hidden cached shelves or allocating a feed. */
internal fun tvAudioItemsEmpty(state: MusicState, visibleTracks: List<AudioTrack>): Boolean {
    if (state.selectedGroup == null && tvAudioGroups(state).isNotEmpty() && state.grouping != MusicGrouping.Discover) return false
    return if (state.grouping == MusicGrouping.Discover) {
        state.discoveryPlayed.isEmpty() && state.discoveryRecent.isEmpty() && state.favoriteTracks.isEmpty()
    } else visibleTracks.isEmpty()
}
