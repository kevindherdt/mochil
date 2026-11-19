package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ExternalDownloadRecoveryTest {
    private data class Candidate(val key: String, val readable: Boolean)

    @Test fun recoveryKeepsOnlyReadableUniqueEntriesMissingFromCurrentState() {
        val candidates = listOf(
            Candidate("podcasts:episode-1", true),
            Candidate("podcasts:episode-1", true),
            Candidate("podcasts:episode-2", false),
            Candidate("music:track-1", true),
            Candidate("audiobooks:chapter-1", true),
        )

        val recovered = recoverableExternalDownloads(
            indexed = candidates,
            currentKeys = setOf("music:track-1"),
            keyOf = Candidate::key,
            isReadable = Candidate::readable,
        )

        assertEquals(listOf("podcasts:episode-1", "audiobooks:chapter-1"), recovered.map(Candidate::key))
    }

    @Test fun emptyOrFullyCurrentIndexDoesNotManufactureRecovery() {
        assertEquals(
            emptyList<Candidate>(),
            recoverableExternalDownloads(
                indexed = listOf(Candidate("podcasts:episode-1", true)),
                currentKeys = setOf("podcasts:episode-1"),
                keyOf = Candidate::key,
                isReadable = Candidate::readable,
            ),
        )
    }

    @Test fun removalIsCompleteOnlyWhenPrivateAndSelectedFolderCopiesAreGone() {
        assertTrue(downloadRemovalComplete(localRemoved = true, externalRemoved = true))
        assertFalse(downloadRemovalComplete(localRemoved = false, externalRemoved = true))
        assertFalse(downloadRemovalComplete(localRemoved = true, externalRemoved = false))
    }

    @Test fun `podcast recovery identity survives a recreated library`() {
        val old = AudioTrack(
            id = "episode-7", libraryId = "podcasts-old", title = "Episode", artist = "Show", album = "Show",
            durationSeconds = 600.0, libraryKind = "podcasts", feedId = "feed-stable",
        )
        val recreated = old.copy(libraryId = "podcasts-new")

        assertEquals(
            offlineAudioRecoveryKey("podcasts-old:episode-7", old),
            offlineAudioRecoveryKey("podcasts-new:episode-7", recreated),
        )
        assertFalse(
            offlineAudioRecoveryKey("podcasts-old:episode-7", old) ==
                offlineAudioRecoveryKey("podcasts-new:episode-7", recreated.copy(feedId = "other-feed")),
        )
        val musicTrack = old.copy(libraryKind = "music")
        assertEquals(musicTrack.identityKey, offlineAudioRecoveryKey("music-old:track", musicTrack))
    }
}
