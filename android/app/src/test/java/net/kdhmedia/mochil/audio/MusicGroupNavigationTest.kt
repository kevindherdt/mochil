package net.kdhmedia.mochil.audio

import org.junit.Assert.*
import org.junit.Test

class MusicGroupNavigationTest {
    private val artist = AudioGroup("Artist", "Artist", 2, "artist", "music")
    private val album = AudioGroup("Album", "Artist", 1, "album", "music")

    @Test fun `back during album loading restores artist albums without a spinner`() {
        val current = MusicState(selectedArtist = artist, selectedGroup = album,
            artistAlbums = listOf(album), groups = listOf(artist), loading = true, loadingMore = true,
            nextOffset = 72, query = "artist", error = "Old failure")
        val parent = current.withClosedMusicGroup()
        assertNull(parent.selectedGroup)
        assertSame(artist, parent.selectedArtist)
        assertEquals(listOf(album), parent.artistAlbums)
        assertEquals(current.groups, parent.groups)
        assertFalse(parent.loading)
        assertFalse(parent.loadingMore)
        assertNull(parent.error)
        assertEquals(72, parent.nextOffset)
        assertEquals("artist", parent.query)
    }

    @Test fun `back during artist loading keeps cached root and pagination cursor`() {
        val current = MusicState(selectedArtist = artist, groups = listOf(artist),
            loading = true, loadingMore = true, nextOffset = 72)
        val parent = current.withClosedMusicArtist()
        assertNull(parent.selectedArtist)
        assertFalse(parent.loading)
        assertFalse(parent.loadingMore)
        assertEquals(current.groups, parent.groups)
        assertEquals(72, parent.nextOffset)
    }

    @Test fun `redundant close does not interrupt root loading`() {
        val root = MusicState(loading = true, loadingMore = true)
        assertSame(root, root.withClosedMusicGroup())
        assertSame(root, root.withClosedMusicArtist())
    }
}
