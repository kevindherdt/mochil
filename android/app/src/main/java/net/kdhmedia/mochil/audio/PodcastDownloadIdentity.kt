package net.kdhmedia.mochil.audio

internal data class OfflinePodcastFeed(val id: String, val title: String)

/**
 * Rebinds a recovered podcast download to the current server catalogue.
 * Feed IDs are authoritative when retained. A recreated subscription may have
 * a different ID, so a unique normalized show title is the only safe fallback.
 */
internal fun reconcileOfflinePodcastTrack(
    track: AudioTrack,
    libraryId: String,
    feeds: List<OfflinePodcastFeed>,
): AudioTrack {
    if (track.libraryKind != "podcasts" || feeds.isEmpty()) return track
    val direct = feeds.singleOrNull { it.id == track.feedId }
    val aliases = listOf(track.album, track.artist).map(::normalizedPodcastShowTitle).filter(String::isNotBlank).toSet()
    val titled = feeds.filter { normalizedPodcastShowTitle(it.title) in aliases }
        .distinctBy(OfflinePodcastFeed::id)
        .singleOrNull()
    val feed = direct ?: titled ?: return track
    return track.copy(libraryId = libraryId, feedId = feed.id, album = feed.title.trim().ifBlank { track.album })
}

/** Exact episode identity wins; metadata fallback must identify one candidate only. */
internal fun reconcileOfflinePodcastEpisode(
    track: AudioTrack,
    libraryId: String,
    feedId: String,
    episodes: List<AudioTrack>,
): AudioTrack {
    // An orphan recovered after reinstall has no kind yet. A unique episode
    // metadata match may classify it; never reinterpret known non-podcast audio.
    if ((track.libraryKind.isNotBlank() && track.libraryKind != "podcasts") || episodes.isEmpty()) return track
    val exact = episodes.filter { it.id == track.id }.singleOrNull()
    val title = normalizedPodcastEpisodeTitle(track.title)
    val metadata = episodes.filter { candidate ->
        normalizedPodcastEpisodeTitle(candidate.title) == title &&
            compatiblePodcastDuration(track.durationSeconds, candidate.durationSeconds) &&
            compatiblePodcastSize(track.sizeBytes, candidate.sizeBytes)
    }.singleOrNull()
    val episode = exact ?: metadata ?: return track
    return episode.copy(libraryId = libraryId, feedId = feedId, libraryKind = "podcasts")
}

private fun normalizedPodcastShowTitle(value: String): String =
    value.trim().lowercase().replace(Regex("\\s+"), " ")

private fun normalizedPodcastEpisodeTitle(value: String): String =
    value.trim().lowercase().replace(Regex("[^\\p{L}\\p{N}]+"), " ").trim()

private fun compatiblePodcastDuration(left: Double, right: Double): Boolean =
    left <= 0.0 || right <= 0.0 || kotlin.math.abs(left - right) <= 3.0

private fun compatiblePodcastSize(left: Long, right: Long): Boolean =
    left <= 0L || right <= 0L || kotlin.math.abs(left - right) <= 16_384L

/**
 * Walk every available episode page so downloads spread across a long-running
 * feed are all reconnected. The cap protects against a broken/cyclic server
 * cursor while still covering feeds much larger than the normal page size.
 */
internal fun shouldContinuePodcastReconciliation(nextCursor: String?, pagesLoaded: Int, pageLimit: Int = 32): Boolean =
    !nextCursor.isNullOrBlank() && pagesLoaded < pageLimit
