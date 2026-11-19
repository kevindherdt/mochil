package net.kdhmedia.mochil.audio

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PodcastDownloadIdentityTest {
    private val recovered = AudioTrack(
        id = "episode-guid", libraryId = "podcasts-old", title = "Episode 7",
        artist = "The Example Show", album = "The Example Show", durationSeconds = 900.0,
        libraryKind = "podcasts", feedId = "old-feed",
    )

    @Test fun `recreated subscription reconnects a recovered episode by unique show title`() {
        val result = reconcileOfflinePodcastTrack(
            recovered,
            "podcasts-new",
            listOf(OfflinePodcastFeed("new-feed", "  THE  Example Show ")),
        )

        assertEquals("podcasts-new", result.libraryId)
        assertEquals("new-feed", result.feedId)
        assertEquals("THE  Example Show", result.album)
    }

    @Test fun `retained feed identity wins even if the display title changed`() {
        val result = reconcileOfflinePodcastTrack(
            recovered,
            "podcasts-new",
            listOf(OfflinePodcastFeed("old-feed", "The Example Show: Remastered")),
        )

        assertEquals("podcasts-new", result.libraryId)
        assertEquals("old-feed", result.feedId)
        assertEquals("The Example Show: Remastered", result.album)
    }

    @Test fun `ambiguous show titles never link a recovered file`() {
        val result = reconcileOfflinePodcastTrack(
            recovered,
            "podcasts-new",
            listOf(
                OfflinePodcastFeed("feed-a", "The Example Show"),
                OfflinePodcastFeed("feed-b", "the example show"),
            ),
        )

        assertEquals(recovered, result)
    }

    @Test fun `unrelated feeds and non podcast audio remain unchanged`() {
        assertEquals(
            recovered,
            reconcileOfflinePodcastTrack(recovered, "podcasts-new", listOf(OfflinePodcastFeed("other", "Another Show"))),
        )
        val music = recovered.copy(libraryKind = "music")
        assertEquals(music, reconcileOfflinePodcastTrack(music, "music-new", listOf(OfflinePodcastFeed("new-feed", "The Example Show"))))
    }

    @Test fun `episode id reconnects even when show and episode titles changed`() {
        val current = recovered.copy(
            libraryId = "podcasts-new", feedId = "new-feed", title = "Episode Seven (Remastered)",
            album = "Renamed Show", artist = "New Publisher", publishedAtMillis = 1234L,
        )

        assertEquals(
            current,
            reconcileOfflinePodcastEpisode(recovered, "podcasts-new", "new-feed", listOf(current)),
        )
    }

    @Test fun `unique normalized title and compatible media facts reconnect a changed episode id`() {
        val current = recovered.copy(
            id = "new-episode-id", libraryId = "podcasts-new", feedId = "new-feed",
            title = " Episode 7! ", sizeBytes = 50_000_000L,
        )
        val stored = recovered.copy(sizeBytes = 50_010_000L)

        assertEquals(
            current,
            reconcileOfflinePodcastEpisode(stored, "podcasts-new", "new-feed", listOf(current)),
        )
    }

    @Test fun `unique episode match classifies an orphaned audio file after reinstall`() {
        val orphan = recovered.copy(
            id = "recovered-file", libraryId = "", feedId = "", libraryKind = "",
            title = "Episode 7", sizeBytes = 50_000_000L,
        )
        val current = recovered.copy(
            id = "rss-episode", libraryId = "podcasts-new", feedId = "new-feed",
            title = "Episode 7!", sizeBytes = 50_000_000L,
        )

        assertEquals(
            current,
            reconcileOfflinePodcastEpisode(orphan, "podcasts-new", "new-feed", listOf(current)),
        )
    }

    @Test fun `ambiguous episode metadata never reconnects a recovered file`() {
        val candidates = listOf(
            recovered.copy(id = "new-a", libraryId = "new", feedId = "feed"),
            recovered.copy(id = "new-b", libraryId = "new", feedId = "feed"),
        )

        val stored = recovered.copy(id = "old")
        assertEquals(stored, reconcileOfflinePodcastEpisode(stored, "new", "feed", candidates))
    }

    @Test fun `reconciliation paging stops on a match end cursor or strict cap`() {
        assertTrue(shouldContinuePodcastReconciliation("next", pagesLoaded = 1))
        assertTrue(shouldContinuePodcastReconciliation("next", pagesLoaded = 1, pageLimit = 8))
        assertFalse(shouldContinuePodcastReconciliation(null, pagesLoaded = 1))
        assertFalse(shouldContinuePodcastReconciliation("next", pagesLoaded = 8, pageLimit = 8))
    }
}
