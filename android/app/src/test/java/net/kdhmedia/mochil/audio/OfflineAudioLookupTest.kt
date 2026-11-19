package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class OfflineAudioLookupTest {
    private val track = AudioTrack("target", "music", "Song", "Artist", "Album", 60.0)

    @Test fun `large library probes storage only for the requested track`() {
        val entries = (1..1000).map { track.copy(id = "$it") } + track
        val probed = mutableListOf<String>()
        val location = resolveOfflineAudioLocation(entries, track, { it }) {
            probed += it.id
            "file:///music/target.flac"
        }
        assertEquals(listOf("target"), probed)
        assertEquals("file:///music/target.flac", location)
    }

    @Test fun `missing identity does not access storage`() {
        val location = resolveOfflineAudioLocation(listOf(track.copy(id = "other")), track, { it }) {
            throw AssertionError("Unrelated file was checked")
        }
        assertNull(location)
    }

    @Test fun `unreadable match does not hide a readable recovered podcast alias`() {
        val requested = track.copy(libraryKind = "podcasts", feedId = "show", libraryId = "new")
        val entries = listOf(requested, requested.copy(libraryId = "old"), requested.copy(libraryId = "older"))
        val probed = mutableListOf<String>()
        val location = resolveOfflineAudioLocation(entries, requested, { it }) {
            probed += it.libraryId
            if (it.libraryId == "old") "content://downloads/episode" else null
        }
        assertEquals(listOf("new", "old"), probed)
        assertEquals("content://downloads/episode", location)
    }
}
