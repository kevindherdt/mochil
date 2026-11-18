package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.MusicPlaylistOverview
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * A playlist has no separate server id, so its name as stored is its identity. Keying the lazy list by a lowercased
 * copy collapsed "Mix" and "mix" — two playlists the server considers distinct — into one key, and a duplicate key in
 * a lazy list throws.
 */
class MusicPlaylistItemKeyTest {
    private fun playlist(name: String, tracks: Int = 2) = MusicPlaylistOverview(name, tracks, emptyList())

    @Test fun `names that differ only in case get distinct keys`() {
        val keys = listOf(playlist("Mix"), playlist("mix"), playlist("MIX")).map(::musicPlaylistItemKey)
        assertEquals(keys.size, keys.toSet().size)
    }

    @Test fun `the key is the stored name, untouched`() {
        assertEquals("Saturday Morning", musicPlaylistItemKey(playlist("Saturday Morning")))
    }
}
