package net.kdhmedia.mochil.audio

import org.junit.Assert.*
import org.junit.Test

class MusicQueueScopeTest {
    private fun track(kind: String) = AudioTrack("one", "library", "Title", "Artist", "Album", 180.0, libraryKind = kind)

    @Test fun `only complete music playback queues can replace saved music`() {
        assertTrue(canPersistMusicPlaybackQueue(listOf(track("music"))))
        assertFalse(canPersistMusicPlaybackQueue(emptyList()))
        for (kind in listOf("audiobooks", "podcasts", "")) {
            assertFalse(canPersistMusicPlaybackQueue(listOf(track(kind))))
            assertFalse(canPersistMusicPlaybackQueue(listOf(track("music"), track(kind))))
            assertFalse(canPersistMusicPlaybackQueue(listOf(track(kind), track("music"))))
        }
    }
}
