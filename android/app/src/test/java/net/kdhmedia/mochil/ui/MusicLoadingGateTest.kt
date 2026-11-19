package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioGroup
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.MusicGrouping
import net.kdhmedia.mochil.audio.MusicState
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The music shelf is the only media screen that replaced whatever was on it with a spinner for any load in flight.
 * Home and the video shelf already keep the rows on screen and only show the spinner when there is nothing to keep;
 * this covers the rule that decides which of those two a load is.
 */
class MusicLoadingGateTest {
    private fun track(id: String) = AudioTrack(id, "music", "Track $id", "Artist", "Album", 200.0, libraryKind = "music")
    private fun group(name: String) = AudioGroup(name, "Artist", 3, "", "music")

    @Test fun `an empty shelf is a load worth the whole screen`() {
        assertTrue(musicShelfIsEmpty(MusicState(loading = true)))
        assertTrue(musicShelfIsEmpty(MusicState(grouping = MusicGrouping.Tracks, loading = true)))
    }

    @Test fun `a shelf with content keeps its rows during a load`() {
        assertFalse(musicShelfIsEmpty(MusicState(tracks = listOf(track("a")))))
        assertFalse(musicShelfIsEmpty(MusicState(groups = listOf(group("Album")))))
    }

    @Test fun `a discovered shelf counts as content, not as an empty one`() {
        assertFalse("Discover draws its own rows and must not be replaced by a spinner", musicShelfIsEmpty(MusicState(discoveryAlbums = listOf(group("Album")))))
        assertFalse(musicShelfIsEmpty(MusicState(discoveryArtists = listOf(group("Artist")))))
        assertFalse(musicShelfIsEmpty(MusicState(discoveryRecent = listOf(track("a")))))
        assertFalse(musicShelfIsEmpty(MusicState(discoveryPlayed = listOf(track("b")))))
    }

    @Test fun `offline and favourite shelves count as content too`() {
        assertFalse(musicShelfIsEmpty(MusicState(offlineTracks = listOf(track("a")), downloadedOnly = true)))
        assertFalse(musicShelfIsEmpty(MusicState(favoriteTracks = listOf(track("b")), favoritesOnly = true)))
    }
}
