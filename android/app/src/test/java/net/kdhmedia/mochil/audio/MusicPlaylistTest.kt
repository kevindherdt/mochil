package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Test

class MusicPlaylistTest {
    @Test fun normalizesWhitespaceWithoutChangingWords() {
        assertEquals("Evening mix", normalizePlaylistName("  Evening   mix  "))
    }

    @Test fun deduplicatesByLibraryAndItemWhileKeepingOrder() {
        val first = AudioTrack("1", "music", "First", "Artist", "Album", 1.0)
        val duplicate = first.copy(title = "Changed display title")
        val otherShelf = first.copy(libraryId = "other")
        assertEquals(listOf(first, otherShelf), playlistTracks(listOf(first, duplicate, otherShelf)))
    }
}
