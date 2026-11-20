package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.identityKey
import org.junit.Assert.assertEquals
import org.junit.Test

class MusicPlaylistArtworkTest {
    private fun track(id: String) = AudioTrack(id, "music", id, "Artist", "Album", 180.0)

    @Test fun offlineStateDeduplicatesTracksAndCountsDownloads() {
        val values = listOf(track("one"), track("two"), track("one"))
        assertEquals(PlaylistOfflineState(downloaded = 1, total = 2), playlistOfflineState(values, setOf(track("one").identityKey)))
    }

    @Test fun offlineSummaryIncludesKnownSizeWithoutPerTrackBadges() {
        val values = listOf(track("one").copy(sizeBytes = 1_048_576), track("two").copy(sizeBytes = 2_097_152))
        assertEquals("2 tracks · 1 offline · 3.0 MB", playlistOfflineSummary(values, PlaylistOfflineState(1, 2)))
    }
}
