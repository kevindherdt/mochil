package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicState
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TvAudioEmptyStateTest {
    private val group = AudioGroup("Album", "Artist", 2, "", "music")
    private val track = AudioTrack("track", "music", "Song", "Artist", "Album", 180.0)

    @Test fun `empty album ignores cached root and discovery content`() {
        val state = MusicState(grouping = MusicGrouping.Albums, selectedGroup = group,
            groups = listOf(group), discoveryPlayed = listOf(track), favoriteTracks = listOf(track))
        assertTrue(tvAudioItemsEmpty(state, emptyList()))
        assertFalse(tvAudioItemsEmpty(state, listOf(track)))
    }

    @Test fun `artist result follows arriving albums rather than cached artists`() {
        val state = MusicState(grouping = MusicGrouping.Artists, selectedArtist = group, groups = listOf(group))
        assertTrue(tvAudioItemsEmpty(state, emptyList()))
        assertFalse(tvAudioItemsEmpty(state.copy(artistAlbums = listOf(group)), emptyList()))
    }

    @Test fun `filtered favorites ignore hidden track and discovery caches`() {
        val state = MusicState(grouping = MusicGrouping.Favorites, tracks = listOf(track),
            favoriteTracks = listOf(track), discoveryRecent = listOf(track))
        assertTrue(tvAudioItemsEmpty(state, emptyList()))
        assertFalse(tvAudioItemsEmpty(state, listOf(track)))
    }

    @Test fun `discovery counts only the track feed rendered on TV`() {
        val state = MusicState(grouping = MusicGrouping.Discover, groups = listOf(group),
            discoveryAlbums = listOf(group), tracks = listOf(track))
        assertTrue(tvAudioItemsEmpty(state, listOf(track)))
        assertFalse(tvAudioItemsEmpty(state.copy(discoveryRecent = listOf(track)), emptyList()))
        assertFalse(tvAudioItemsEmpty(state.copy(favoriteTracks = listOf(track)), emptyList()))
    }
}
