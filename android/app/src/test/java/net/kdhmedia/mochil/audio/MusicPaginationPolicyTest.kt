package net.kdhmedia.mochil.audio

import org.junit.Assert.*
import org.junit.Test

class MusicPaginationPolicyTest {
    private val root = MusicState(grouping = MusicGrouping.Albums, nextOffset = 72)
    private val group = AudioGroup("Album", "Artist", 1, "one", "music")

    @Test fun `only ready online root accepts its continuation`() {
        assertTrue(root.canLoadMoreMusic())
        assertFalse(root.copy(loading = true).canLoadMoreMusic())
        assertFalse(root.copy(loadingMore = true).canLoadMoreMusic())
        assertFalse(root.copy(nextOffset = null).canLoadMoreMusic())
    }

    @Test fun `root cursor cannot page inside album or artist detail`() {
        assertFalse(root.copy(selectedGroup = group).canLoadMoreMusic())
        assertFalse(root.copy(selectedArtist = group).canLoadMoreMusic())
        assertTrue(root.copy(selectedGroup = group, loading = true).withClosedMusicGroup().canLoadMoreMusic())
    }

    @Test fun `local and favorite filters do not request unrelated root pages`() {
        assertFalse(root.copy(downloadedOnly = true).canLoadMoreMusic())
        assertFalse(root.copy(favoritesOnly = true).canLoadMoreMusic())
        assertFalse(root.copy(offlineOnly = true).canLoadMoreMusic())
    }

    @Test fun `track continuation requires its own cursor not a group offset`() {
        assertFalse(root.copy(grouping = MusicGrouping.Tracks).canLoadMoreMusic())
        val tracks = root.copy(grouping = MusicGrouping.Tracks, nextCursor = "next", nextOffset = null)
        assertTrue(tracks.canLoadMoreMusic())
        assertFalse(tracks.copy(selectedGroup = group).canLoadMoreMusic())
    }
}
