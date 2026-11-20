package net.kdhmedia.mochil.podcast

import net.kdhmedia.mochil.audio.AudioTrack
import java.time.Instant
import java.time.ZoneOffset

enum class PodcastLayout { Covers, List }
enum class PodcastPlayedFilter { All, Unplayed, Played, Downloaded }
enum class PodcastEpisodeSort(val label: String, val queryValue: String) {
    Newest("Newest first", "newest"),
    Oldest("Oldest first", "oldest"),
    Title("Title A–Z", "title"),
}

data class PodcastShow(
    val id: String,
    val title: String,
    val description: String,
    val episodeCount: Int,
    val policy: PodcastPolicy = PodcastPolicy(),
    val lastError: String = "",
)

/** What to say under a show whose feed could not be refreshed; its saved episodes still play. */
fun PodcastShow.feedNote(): String? = "Feed refresh failed · saved episodes are still available".takeIf { lastError.isNotBlank() }

data class PodcastPolicy(val autoDownload: Boolean = false, val retainDownloads: Int = 0, val deletePlayedAfterDays: Int = 0)

data class PodcastShowPage(val items: List<PodcastShow>, val nextOffset: Int?)
data class PodcastEpisodePage(val items: List<AudioTrack>, val nextCursor: String?)
data class PodcastDiscoveryResult(val title: String, val author: String, val artworkPath: String, val feedUrl: String, val episodeCount: Int, val genre: String)

internal fun isCrediblePodcastDate(timestampMillis: Long, nowMillis: Long = System.currentTimeMillis()): Boolean {
    if (timestampMillis <= 0 || timestampMillis > nowMillis + 366L * 24 * 60 * 60 * 1000) return false
    return runCatching { Instant.ofEpochMilli(timestampMillis).atZone(ZoneOffset.UTC).year >= 1900 }.getOrDefault(false)
}

internal fun mergePodcastShows(existing: List<PodcastShow>, incoming: List<PodcastShow>, firstPage: Boolean): List<PodcastShow> =
    (if (firstPage) incoming else existing + incoming).distinctBy(PodcastShow::id)

internal fun mergePodcastEpisodes(existing: List<AudioTrack>, incoming: List<AudioTrack>, firstPage: Boolean): List<AudioTrack> =
    (if (firstPage) incoming else existing + incoming).distinctBy(AudioTrack::id)

internal fun sortPodcastEpisodes(episodes: List<AudioTrack>, sort: PodcastEpisodeSort): List<AudioTrack> {
    val titleOrder = compareBy<AudioTrack>({ it.title.lowercase() }, AudioTrack::id)
    return when (sort) {
        PodcastEpisodeSort.Title -> episodes.sortedWith(titleOrder)
        PodcastEpisodeSort.Oldest -> episodes.sortedWith(compareBy<AudioTrack> { it.publishedAtMillis }.then(titleOrder))
        PodcastEpisodeSort.Newest -> episodes.sortedWith(compareByDescending<AudioTrack> { it.publishedAtMillis }.then(titleOrder))
    }
}

data class PodcastState(
    val libraryId: String = "",
    val layout: PodcastLayout = PodcastLayout.Covers,
    val query: String = "",
    val playedFilter: PodcastPlayedFilter = PodcastPlayedFilter.All,
    val episodeSort: PodcastEpisodeSort = PodcastEpisodeSort.Newest,
    val shows: List<PodcastShow> = emptyList(),
    val selectedShow: PodcastShow? = null,
    val episodes: List<AudioTrack> = emptyList(),
    val loading: Boolean = false,
    val loadingMore: Boolean = false,
    val nextOffset: Int? = null,
    val nextCursor: String? = null,
    val error: String? = null,
    val downloadedIds: Set<String> = emptySet(),
    val downloadingId: String? = null,
    val resolvingPlaybackId: String? = null,
    val updatingPlayedIds: Set<String> = emptySet(),
    val message: String? = null,
    val messageIsError: Boolean = false,
    val subscriptionBusy: Boolean = false,
    val discoveryQuery: String = "",
    val discoveryResults: List<PodcastDiscoveryResult> = emptyList(),
    val discoverySearching: Boolean = false,
    val discoveryError: String? = null,
    val offlineOnly: Boolean = false,
)

internal fun downloadedPodcastShows(tracks: List<AudioTrack>): List<PodcastShow> = tracks
    .filter { it.libraryKind == "podcasts" && it.feedId.isNotBlank() }
    .groupBy(AudioTrack::feedId)
    .map { (feedId, episodes) -> PodcastShow(feedId, episodes.first().album.ifBlank { episodes.first().artist.ifBlank { "Downloaded podcast" } }, "Available offline", episodes.size) }
    .sortedBy { it.title.lowercase() }
