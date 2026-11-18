package net.kdhmedia.mochil.audio

import org.junit.Assert.*
import org.junit.Test

class MusicBucketNavigationTest {
    private val root = MusicState(libraryId = "music", grouping = MusicGrouping.Artists,
        navigation = listOf(AudioNavigationBucket("M", 42, 8)), loadingMore = true, nextOffset = 60,
        query = "artist", error = "Old failure")

    @Test fun `jump replaces paging state while retaining query and navigation`() {
        val jump = root.musicBucketJump("M")!!
        assertTrue(root.supportsMusicBucketNavigation())
        assertEquals(42, jump.offset)
        assertTrue(jump.state.loading)
        assertFalse(jump.state.loadingMore)
        assertNull(jump.state.nextOffset)
        assertNull(jump.state.error)
        assertEquals(root.query, jump.state.query)
        assertEquals(root.navigation, jump.state.navigation)
        assertEquals("M", jump.state.bucket)
    }

    @Test fun `selecting active letter resets to start without losing browse mode`() {
        val jump = root.copy(bucket = "M").musicBucketJump("M")!!
        assertEquals(0, jump.offset)
        assertEquals("", jump.state.bucket)
        assertEquals(MusicGrouping.Artists, jump.state.grouping)
    }

    @Test fun `root offsets cannot be applied to details or local filtered views`() {
        val group = AudioGroup("Album", "Artist", 1, "", "music")
        for (state in listOf(root.copy(selectedGroup = group), root.copy(selectedArtist = group),
            root.copy(downloadedOnly = true), root.copy(favoritesOnly = true), root.copy(offlineOnly = true),
            root.copy(grouping = MusicGrouping.Discover), root.copy(libraryKind = "audiobooks"))) {
            assertNull(state.musicBucketJump("M"))
            assertFalse(state.supportsMusicBucketNavigation())
        }
        assertNull(root.musicBucketJump("Z"))
        assertNull(root.copy(navigation = listOf(AudioNavigationBucket("M", -1, 1))).musicBucketJump("M"))
    }
}
