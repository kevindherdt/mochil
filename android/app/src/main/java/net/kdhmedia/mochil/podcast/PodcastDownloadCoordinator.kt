package net.kdhmedia.mochil.podcast

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.audio.OfflineAudioStore
import net.kdhmedia.mochil.audio.OfflinePodcastFeed
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.audio.shouldContinuePodcastReconciliation
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

/** Owns device-local podcast files, recovery, retention, and automatic downloads. */
internal class PodcastDownloadCoordinator(
    private val context: Context,
    private val client: PodcastClient,
    private val credential: DeviceCredential?,
) {
    private val store = OfflineAudioStore(context)
    private val policies = PodcastDevicePolicyStore(context)
    private val activeAutomaticFeeds = mutableSetOf<String>()

    fun ids(): Set<String> = store.ids()
    fun tracks(): List<AudioTrack> = store.tracks()
    fun contains(track: AudioTrack): Boolean = store.contains(track)
    fun playbackUrl(track: AudioTrack): String? = store.playbackUrl(track)
    fun offlineShows(): List<PodcastShow> = downloadedPodcastShows(store.tracks())

    fun reconcileFeeds(libraryId: String, shows: List<PodcastShow>) {
        store.reconcilePodcastFeeds(libraryId, shows.map { OfflinePodcastFeed(it.id, it.title) })
    }

    fun reconcileEpisodePage(libraryId: String, showId: String, episodes: List<AudioTrack>) {
        store.reconcilePodcastEpisodes(libraryId, showId, episodes)
    }

    fun downloadedIds(episodes: List<AudioTrack>): Set<String> =
        store.ids() + episodes.asSequence().filter(store::contains).map(AudioTrack::podcastDownloadKey)

    fun policy(show: PodcastShow): PodcastDevicePolicy = policies.get(show.id)

    fun updatePolicy(show: PodcastShow, policy: PodcastDevicePolicy) {
        policies.put(show.id, policy)
    }

    fun toggle(track: AudioTrack): Result<Unit> = runCatching {
        if (store.contains(track)) {
            if (!store.remove(track)) error("Could not remove this download.")
        } else {
            val saved = credential ?: error("This device is no longer connected.")
            store.download(saved, track, client.streamUrl(saved, track))
        }
    }

    fun resolveDownloaded(libraryId: String, show: PodcastShow, sort: PodcastEpisodeSort): PodcastDownloadedEpisodes {
        val saved = credential
        if (saved == null) return downloadedSnapshot(libraryId, show, sort)
        store.recoverExternalDownloads()
        var cursor: String? = null
        var pages = 0
        var matched = 0
        val serverDownloads = mutableListOf<AudioTrack>()
        val visitedCursors = mutableSetOf<String>()
        val result = runCatching {
            do {
                val page = client.episodes(saved, libraryId, show, PodcastPlayedFilter.All, sort, cursor)
                matched += store.reconcilePodcastEpisodes(libraryId, show.id, page.items)
                serverDownloads += page.items.filter(AudioTrack::serverDownloaded)
                cursor = page.nextCursor
                pages++
            } while (
                shouldContinuePodcastReconciliation(cursor, pages, MAX_RECONCILIATION_PAGES) &&
                visitedCursors.add(cursor.orEmpty())
            )
        }
        PerformanceDiagnostics.record("network", "reconcile_podcast_downloads", outcome = if (result.isSuccess) "ok" else "error")
        return if (result.isSuccess) downloadedSnapshot(libraryId, show, sort, matched, serverDownloads.distinctBy(AudioTrack::id))
        else downloadedSnapshot(libraryId, show, sort)
    }

    fun applyAutomaticPolicy(libraryId: String, show: PodcastShow, newest: List<AudioTrack>): Result<Unit>? {
        val policy = policies.get(show.id)
        val saved = credential ?: return null
        if (!policy.autoDownload || newest.isEmpty()) return null
        if (policy.unmeteredOnly && !hasUnmeteredNetwork()) return null
        if (!synchronized(activeAutomaticFeeds) { activeAutomaticFeeds.add(show.id) }) return null
        return runCatching {
            val local = store.tracks().filter { it.libraryKind == "podcasts" && it.feedId == show.id }
                .map { it.copy(libraryId = libraryId) }
            val localByKey = local.associateBy(AudioTrack::podcastDownloadKey)
            val newestByKey = newest.associateBy(AudioTrack::podcastDownloadKey)
            val plan = podcastDeviceDownloadPlan(newest.map(AudioTrack::podcastDownloadKey), localByKey.keys, policy.retainDownloads)
            plan.remove.mapNotNull(localByKey::get).forEach { if (!store.remove(it)) error("Could not apply local podcast retention.") }
            plan.download.mapNotNull(newestByKey::get).forEach { store.download(saved, it, client.streamUrl(saved, it)) }
        }.also { synchronized(activeAutomaticFeeds) { activeAutomaticFeeds.remove(show.id) } }
    }

    private fun downloadedSnapshot(
        libraryId: String,
        show: PodcastShow,
        sort: PodcastEpisodeSort,
        matched: Int = 0,
        serverDownloads: List<AudioTrack> = emptyList(),
    ): PodcastDownloadedEpisodes {
        val local = store.tracks().filter { it.libraryKind == "podcasts" && it.feedId == show.id }
            .map { it.copy(libraryId = libraryId) }
        val available = (serverDownloads + local).distinctBy(AudioTrack::id)
        val episodes = when (sort) {
            PodcastEpisodeSort.Title -> available.sortedWith(compareBy(String.CASE_INSENSITIVE_ORDER, AudioTrack::title))
            PodcastEpisodeSort.Oldest -> available.sortedBy(AudioTrack::publishedAtMillis)
            PodcastEpisodeSort.Newest -> available.sortedByDescending(AudioTrack::publishedAtMillis)
        }
        val recoveredCount = store.tracks().count { it.libraryKind == "podcasts" }
        val unclassifiedCount = store.tracks().count { it.libraryKind.isBlank() }
        val diagnostic = when {
            matched > 0 -> "podcast_downloads_device_match"
            serverDownloads.isNotEmpty() -> "podcast_downloads_server_match"
            recoveredCount > 0 || unclassifiedCount > 0 -> "podcast_downloads_unmatched"
            else -> "podcast_downloads_empty"
        }
        return PodcastDownloadedEpisodes(episodes, downloadedIds(episodes), matched, serverDownloads.size, recoveredCount, unclassifiedCount, diagnostic)
    }

    private fun hasUnmeteredNetwork(): Boolean {
        val manager = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val network = manager.activeNetwork ?: return false
        return manager.getNetworkCapabilities(network)?.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED) == true
    }

    private companion object { const val MAX_RECONCILIATION_PAGES = 32 }
}

internal data class PodcastDownloadedEpisodes(
    val episodes: List<AudioTrack>,
    val downloadedIds: Set<String>,
    val matched: Int,
    val serverDownloadCount: Int,
    val recoveredCount: Int,
    val unclassifiedCount: Int,
    val diagnostic: String,
)

private fun AudioTrack.podcastDownloadKey() = identityKey
