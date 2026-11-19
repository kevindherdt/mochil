package net.kdhmedia.mochil.podcast

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PodcastStateTest {
    @Test fun implausiblePodcastDatesAreNotDisplayed() {
        val now = java.time.Instant.parse("2026-08-27T12:00:00Z").toEpochMilli()
        assertFalse(isCrediblePodcastDate(0, now))
        assertFalse(isCrediblePodcastDate(java.time.Instant.parse("0001-01-01T00:00:00Z").toEpochMilli(), now))
        assertFalse(isCrediblePodcastDate(java.time.Instant.parse("2030-01-01T00:00:00Z").toEpochMilli(), now))
        assertTrue(isCrediblePodcastDate(java.time.Instant.parse("2024-08-27T12:00:00Z").toEpochMilli(), now))
    }

    @Test fun episodeSortUsesStableServerQueryValues() {
        assertEquals(listOf("newest", "oldest", "title"), PodcastEpisodeSort.entries.map(PodcastEpisodeSort::queryValue))
    }

    @Test fun episodeSortIsDeterministicAfterPagedResultsMerge() {
        val newest = AudioTrack("new", "podcasts", "Episode 10", "Show", "Show", 60.0, publishedAtMillis = 30)
        val oldest = AudioTrack("old", "podcasts", "Episode 1", "Show", "Show", 60.0, publishedAtMillis = 10)
        val middle = AudioTrack("middle", "podcasts", "Episode 2", "Show", "Show", 60.0, publishedAtMillis = 20)
        val input = listOf(middle, newest, oldest)
        assertEquals(listOf("old", "middle", "new"), sortPodcastEpisodes(input, PodcastEpisodeSort.Oldest).map(AudioTrack::id))
        assertEquals(listOf("new", "middle", "old"), sortPodcastEpisodes(input, PodcastEpisodeSort.Newest).map(AudioTrack::id))
    }
    @Test fun continuationPagesDoNotDuplicateShowsOrEpisodes() {
        val show = PodcastShow("show", "Show", "", 2)
        assertEquals(listOf(show), mergePodcastShows(listOf(show), listOf(show), false))
        val episode = AudioTrack("episode", "podcasts", "Episode", "Show", "Show", 60.0, libraryKind = "podcasts")
        assertEquals(listOf(episode), mergePodcastEpisodes(listOf(episode), listOf(episode), false))
    }

    @Test fun firstPageReplacesStaleResults() {
        val old = PodcastShow("old", "Old", "", 1)
        val fresh = PodcastShow("fresh", "Fresh", "", 1)
        assertEquals(listOf(fresh), mergePodcastShows(listOf(old), listOf(fresh), true))
    }

    @Test fun episodeResponsesMatchTheExactVisibleContext() {
        val show = PodcastShow("show", "Show", "", 1)
        val request = PodcastState(libraryId = "podcasts", selectedShow = show, playedFilter = PodcastPlayedFilter.Unplayed)
        assertTrue(request.matchesEpisodeRequest(request, show))
        assertFalse(request.copy(libraryId = "other").matchesEpisodeRequest(request, show))
        assertFalse(request.copy(selectedShow = PodcastShow("other", "Other", "", 1)).matchesEpisodeRequest(request, show))
        assertFalse(request.copy(playedFilter = PodcastPlayedFilter.Played).matchesEpisodeRequest(request, show))
        assertFalse(request.copy(episodeSort = PodcastEpisodeSort.Oldest).matchesEpisodeRequest(request, show))
    }

    @Test fun deviceDownloadPlanAddsNewestAndRemovesOutsideRetention() {
        val plan = podcastDeviceDownloadPlan(listOf("new", "middle", "old"), setOf("middle", "stale"), 2)
        assertEquals(setOf("new"), plan.download)
        assertEquals(setOf("stale"), plan.remove)
    }

    @Test fun offlineEpisodesGroupByStableFeedIdentity() {
        val tracks = listOf(
            AudioTrack("one", "podcasts", "One", "Show", "Show", 60.0, libraryKind = "podcasts", feedId = "feed"),
            AudioTrack("two", "podcasts", "Two", "Show", "Show", 60.0, libraryKind = "podcasts", feedId = "feed"),
        )
        assertEquals(listOf(PodcastShow("feed", "Show", "Available offline", 2)), downloadedPodcastShows(tracks))
    }

    @Test fun downloadedEpisodeFeedbackPrefersConcreteRecoveryOutcomes() {
        val episode = AudioTrack("one", "podcasts", "One", "Show", "Show", 60.0, libraryKind = "podcasts", feedId = "feed")
        fun result(matched: Int = 0, server: Int = 0, recovered: Int = 0, unclassified: Int = 0, episodes: List<AudioTrack> = emptyList()) =
            PodcastDownloadedEpisodes(episodes, emptySet(), matched, server, recovered, unclassified, "diagnostic")

        assertEquals("Reconnected 1 downloaded episode.", downloadedEpisodeMessage(result(matched = 1, server = 2)))
        assertEquals("Found 2 episodes already stored on the server.", downloadedEpisodeMessage(result(server = 2)))
        assertEquals("Recovered podcast files were found, but none matched this RSS show.", downloadedEpisodeMessage(result(recovered = 1)))
        assertEquals("Recovered audio files were found, but none matched this RSS show's episode metadata.", downloadedEpisodeMessage(result(unclassified = 1)))
        assertEquals(null, downloadedEpisodeMessage(result(recovered = 1, episodes = listOf(episode))))
    }
}
