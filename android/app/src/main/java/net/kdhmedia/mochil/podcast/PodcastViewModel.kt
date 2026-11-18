package net.kdhmedia.mochil.podcast

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.catalogue.BrowseLayoutStore
import net.kdhmedia.mochil.audio.identityKey
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics

class PodcastViewModel(
    application: Application,
    // The downloaded-ids lookup does a disk/Binder round trip; tests inject a dispatcher they can
    // pump so the "never computed inline" guarantee is asserted instead of sampled.
    private val io: kotlinx.coroutines.CoroutineDispatcher = kotlinx.coroutines.Dispatchers.IO,
) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = PodcastClient()
    private val downloads = PodcastDownloadCoordinator(application, client, credential)
    private val playbackResolution = PodcastPlaybackResolution()
    private val preferences = application.getSharedPreferences("podcast_view", 0)
    private val layouts = BrowseLayoutStore(application)
    private val mutableState = MutableStateFlow(PodcastState(
        layout = if (layouts.isList()) PodcastLayout.List else PodcastLayout.Covers,
        episodeSort = runCatching { PodcastEpisodeSort.valueOf(preferences.getString("episode_sort", PodcastEpisodeSort.Newest.name)!!) }.getOrDefault(PodcastEpisodeSort.Newest),
    ))
    val state: StateFlow<PodcastState> = mutableState.asStateFlow()
    private val layoutListener = layouts.listen { list -> mutableState.value = mutableState.value.copy(layout = if (list) PodcastLayout.List else PodcastLayout.Covers) }

    init {
        refreshDownloadedIds()
    }

    fun open(libraryId: String) {
        if (libraryId == mutableState.value.libraryId) return
        cancelPlaybackResolution()
        mutableState.value = PodcastState(libraryId = libraryId, layout = mutableState.value.layout, episodeSort = mutableState.value.episodeSort, loading = true)
        refreshDownloadedIds()
        loadShows(0)
    }

    fun setLayout(layout: PodcastLayout) {
        layouts.setList(layout == PodcastLayout.List)
        mutableState.value = mutableState.value.copy(layout = layout)
    }

    fun refreshOfflineDownloads() {
        val current = mutableState.value
        viewModelScope.launch {
            val ids = withContext(Dispatchers.IO) {
                downloads.reconcileFeeds(current.libraryId, current.shows)
                downloadedIds(current.episodes)
            }
            mutableState.value = mutableState.value.copy(downloadedIds = ids)
            if (current.playedFilter == PodcastPlayedFilter.Downloaded) {
                current.selectedShow?.let(::loadDownloadedEpisodes)
            }
        }
    }

    fun search(query: String) {
        val normalized = query.trim()
        val current = mutableState.value
        if (normalized == current.query) return
        mutableState.value = current.copy(query = normalized, shows = emptyList(), nextOffset = null, loading = true, error = null)
        loadShows(0)
    }

    fun openShow(show: PodcastShow) {
        cancelPlaybackResolution()
        mutableState.value = mutableState.value.copy(selectedShow = show, episodes = emptyList(), nextCursor = null, loading = true, error = null)
        if (mutableState.value.offlineOnly || mutableState.value.playedFilter == PodcastPlayedFilter.Downloaded) {
            mutableState.value = mutableState.value.copy(playedFilter = PodcastPlayedFilter.Downloaded)
            loadDownloadedEpisodes(show)
        } else loadEpisodes(null)
    }

    fun closeShow() {
        cancelPlaybackResolution()
        mutableState.value = mutableState.value.copy(selectedShow = null, episodes = emptyList(), nextCursor = null, error = null)
    }

    fun setPlayedFilter(filter: PodcastPlayedFilter) {
        val current = mutableState.value
        if (filter == current.playedFilter) return
        mutableState.value = current.copy(playedFilter = filter, episodes = emptyList(), nextCursor = null, loading = true, error = null)
        val show = current.selectedShow ?: return
        if (filter == PodcastPlayedFilter.Downloaded) loadDownloadedEpisodes(show) else loadEpisodes(null)
    }

    fun setEpisodeSort(sort: PodcastEpisodeSort) {
        val current = mutableState.value
        if (sort == current.episodeSort) return
        preferences.edit().putString("episode_sort", sort.name).apply()
        mutableState.value = current.copy(episodeSort = sort, episodes = emptyList(), nextCursor = null, loading = true, error = null)
        val show = current.selectedShow ?: return
        if (current.playedFilter == PodcastPlayedFilter.Downloaded) loadDownloadedEpisodes(show) else loadEpisodes(null)
    }

    fun loadMore() {
        val current = mutableState.value
        if (current.loadingMore) return
        if (current.selectedShow != null) {
            current.nextCursor ?: return
            mutableState.value = current.copy(loadingMore = true, error = null)
            loadEpisodes(current.nextCursor)
        } else {
            val offset = current.nextOffset ?: return
            mutableState.value = current.copy(loadingMore = true, error = null)
            loadShows(offset)
        }
    }

    fun retry() {
        val current = mutableState.value
        mutableState.value = current.copy(loading = current.shows.isEmpty() && current.episodes.isEmpty(), loadingMore = false, error = null)
        if (current.selectedShow != null && current.playedFilter == PodcastPlayedFilter.Downloaded) loadDownloadedEpisodes(current.selectedShow)
        else if (current.selectedShow != null) loadEpisodes(if (current.episodes.isEmpty()) null else current.nextCursor)
        else loadShows(if (current.shows.isEmpty()) 0 else current.nextOffset ?: 0)
    }

    fun toggleDownload(episode: AudioTrack) {
        val key = episode.key()
        if (mutableState.value.downloadingId != null) return
        mutableState.value = mutableState.value.copy(downloadingId = key, message = null, messageIsError = false)
        viewModelScope.launch {
            val (result, ids) = withContext(Dispatchers.IO) { downloads.toggle(episode) to downloadedIds(mutableState.value.episodes) }
            val current = mutableState.value
            mutableState.value = current.copy(
                downloadingId = null, downloadedIds = ids, message = result.exceptionOrNull()?.message, messageIsError = result.isFailure,
                episodes = if (current.playedFilter == PodcastPlayedFilter.Downloaded) current.episodes.filter { it.key() in ids } else current.episodes,
            )
        }
    }

    fun setPlayed(episode: AudioTrack, played: Boolean) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val key = episode.key()
        if (key in mutableState.value.updatingPlayedIds) return
        mutableState.value = mutableState.value.copy(updatingPlayedIds = mutableState.value.updatingPlayedIds + key, message = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.setPlayed(saved, episode, played) } }
                .onSuccess {
                    val current = mutableState.value
                    val updated = episode.copy(finished = played, resumePositionSeconds = if (played) episode.durationSeconds else 0.0)
                    val visible = current.episodes.map { if (it.id == episode.id) updated else it }.filter { item ->
                        when (current.playedFilter) {
                            PodcastPlayedFilter.Unplayed -> !item.finished
                            PodcastPlayedFilter.Played -> item.finished
                            else -> true
                        }
                    }
                    mutableState.value = current.copy(
                        episodes = visible,
                        updatingPlayedIds = current.updatingPlayedIds - key,
                        nextCursor = null,
                        message = if (played) "Marked ${episode.title} as played." else "Marked ${episode.title} as unplayed.",
                        messageIsError = false,
                    )
                    if (current.playedFilter != PodcastPlayedFilter.Downloaded) loadEpisodes(null)
                }
                .onFailure {
                    mutableState.value = mutableState.value.copy(
                        updatingPlayedIds = mutableState.value.updatingPlayedIds - key,
                        message = "Could not update this episode. Try again.",
                        messageIsError = true,
                    )
                }
        }
    }

    fun resolveForPlayback(episode: AudioTrack, onResolved: (AudioTrack) -> Unit) {
        val saved = credential ?: return onResolved(episode)
        val ticket = playbackResolution.begin()
        mutableState.value = mutableState.value.copy(resolvingPlaybackId = episode.identityKey, message = null)
        viewModelScope.launch {
            val result = runCatching { withContext(Dispatchers.IO) { client.latestProgress(saved, episode) } }
            if (!playbackResolution.isCurrent(ticket)) return@launch
            val current = mutableState.value
            mutableState.value = current.copy(
                resolvingPlaybackId = null,
                message = result.exceptionOrNull()?.let { "Could not refresh progress. Playing from the last known position." },
                messageIsError = result.isFailure,
            )
            onResolved(result.getOrDefault(episode))
        }
    }

    fun cancelPlaybackResolution() {
        playbackResolution.cancel()
        if (mutableState.value.resolvingPlaybackId != null) {
            mutableState.value = mutableState.value.copy(resolvingPlaybackId = null)
        }
    }

    fun subscribe(feedUrl: String) {
        val url = feedUrl.trim()
        if (url.isBlank() || mutableState.value.subscriptionBusy) return
        val saved = credential ?: return fail("This device is no longer connected.")
        mutableState.value = mutableState.value.copy(subscriptionBusy = true, message = null, messageIsError = false)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.subscribe(saved, url) } }
                .onSuccess {
                    mutableState.value = mutableState.value.copy(subscriptionBusy = false, shows = emptyList(), nextOffset = null, loading = true, message = "Podcast subscribed.", messageIsError = false)
                    loadShows(0)
                }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(subscriptionBusy = false, message = subscriptionFailure(failure), messageIsError = true) }
        }
    }

    fun discover(query: String) {
        val normalized = query.trim()
        if (normalized.isBlank() || mutableState.value.discoverySearching) return
        val saved = credential ?: return fail("This device is no longer connected.")
        mutableState.value = mutableState.value.copy(discoveryQuery = normalized, discoverySearching = true, discoveryError = null)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.discover(saved, normalized) } }
                .onSuccess { results ->
                    if (mutableState.value.discoveryQuery == normalized) mutableState.value = mutableState.value.copy(discoveryResults = results, discoverySearching = false, discoveryError = null)
                }
                .onFailure {
                    if (mutableState.value.discoveryQuery == normalized) mutableState.value = mutableState.value.copy(discoveryResults = emptyList(), discoverySearching = false, discoveryError = "Podcast search is temporarily unavailable.")
                }
        }
    }

    fun clearDiscovery() {
        mutableState.value = mutableState.value.copy(discoveryQuery = "", discoveryResults = emptyList(), discoverySearching = false, discoveryError = null)
    }

    fun unsubscribe(show: PodcastShow) {
        if (mutableState.value.subscriptionBusy) return
        val saved = credential ?: return fail("This device is no longer connected.")
        mutableState.value = mutableState.value.copy(subscriptionBusy = true, message = null, messageIsError = false)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.unsubscribe(saved, show.id) } }
                .onSuccess {
                    val current = mutableState.value
                    mutableState.value = current.copy(subscriptionBusy = false, shows = current.shows.filterNot { it.id == show.id }, message = "${show.title} unsubscribed.", messageIsError = false)
                }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(subscriptionBusy = false, message = subscriptionFailure(failure), messageIsError = true) }
        }
    }

    fun refreshShow(show: PodcastShow) {
        if (mutableState.value.subscriptionBusy) return
        val saved = credential ?: return fail("This device is no longer connected.")
        mutableState.value = mutableState.value.copy(subscriptionBusy = true, message = null, messageIsError = false)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.refresh(saved, show.id) } }
                .onSuccess {
                    val current = mutableState.value
                    mutableState.value = current.copy(subscriptionBusy = false, shows = emptyList(), nextOffset = null, loading = true, message = "${show.title} refreshed.", messageIsError = false)
                    loadShows(0)
                }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(subscriptionBusy = false, message = subscriptionFailure(failure), messageIsError = true) }
        }
    }

    fun updatePolicy(show: PodcastShow, policy: PodcastPolicy) {
        if (mutableState.value.subscriptionBusy) return
        val saved = credential ?: return fail("This device is no longer connected.")
        mutableState.value = mutableState.value.copy(subscriptionBusy = true, message = null, messageIsError = false)
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.updatePolicy(saved, show.id, policy) } }
                .onSuccess {
                    val current = mutableState.value
                    mutableState.value = current.copy(
                        subscriptionBusy = false,
                        shows = current.shows.map { if (it.id == show.id) it.copy(policy = policy) else it },
                        message = "Download policy saved.", messageIsError = false,
                    )
                }
                .onFailure { failure -> mutableState.value = mutableState.value.copy(subscriptionBusy = false, message = subscriptionFailure(failure), messageIsError = true) }
        }
    }

    fun devicePolicy(show: PodcastShow): PodcastDevicePolicy = downloads.policy(show)

    fun updateDevicePolicy(show: PodcastShow, policy: PodcastDevicePolicy) {
        downloads.updatePolicy(show, policy)
        mutableState.value = mutableState.value.copy(message = "Device download policy saved.", messageIsError = false)
        if (mutableState.value.selectedShow?.id == show.id && mutableState.value.episodes.isNotEmpty()) reconcileDeviceDownloads(show, mutableState.value.episodes)
    }

    // downloads.playbackUrl() checks disk and, for externally-stored episodes, makes a real Binder
    // IPC via DownloadLocationStore to the storage provider -- never call it off Dispatchers.IO.
    suspend fun playbackUrl(episode: AudioTrack): String =
        withContext(Dispatchers.IO) { downloads.playbackUrl(episode) } ?: credential?.let { client.streamUrl(it, episode) }.orEmpty()
    fun artworkUrl(episode: AudioTrack): String = credential?.let { client.episodeArtworkUrl(it, episode) }.orEmpty()
    fun artworkUrl(show: PodcastShow): String = credential?.let { client.showArtworkUrl(it, show) }.orEmpty()
    fun artworkUrl(result: PodcastDiscoveryResult): String = credential?.let { client.discoveryArtworkUrl(it, result) }.orEmpty()

    private fun loadShows(offset: Int) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val request = mutableState.value
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                val page = PerformanceDiagnostics.measure("network", "load_podcast_shows") { client.shows(saved, offset, request.query) }
                downloads.reconcileFeeds(request.libraryId, page.items)
                page
            } }
                .onSuccess { page ->
                    val current = mutableState.value
                    if (current.libraryId != request.libraryId || current.query != request.query || current.selectedShow != null) return@onSuccess
                    mutableState.value = current.copy(shows = mergePodcastShows(current.shows, page.items, offset == 0), nextOffset = page.nextOffset, loading = false, loadingMore = false, offlineOnly = false)
                }
                .onFailure {
                    val current = mutableState.value
                    if (current.libraryId != request.libraryId || current.query != request.query || current.selectedShow != null) return@onFailure
                    val offline = if (offset == 0 && current.query.isBlank()) downloads.offlineShows() else emptyList()
                    if (offline.isNotEmpty()) mutableState.value = current.copy(shows = offline, nextOffset = null, loading = false, loadingMore = false, error = null, message = "Showing downloaded podcasts offline.", messageIsError = false, offlineOnly = true)
                    else fail("Could not load podcast shows.")
                }
        }
    }

    private fun loadEpisodes(cursor: String?) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val request = mutableState.value
        val show = request.selectedShow ?: return
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                val page = PerformanceDiagnostics.measure("network", "load_podcast_episodes") { client.episodes(saved, request.libraryId, show, request.playedFilter, request.episodeSort, cursor) }
                downloads.reconcileEpisodePage(request.libraryId, show.id, page.items)
                page
            } }
                .onSuccess { page ->
                    val current = mutableState.value
                    if (!current.matchesEpisodeRequest(request, show)) return@onSuccess
                    val episodes = sortPodcastEpisodes(mergePodcastEpisodes(current.episodes, page.items, cursor == null), current.episodeSort)
                    mutableState.value = current.copy(episodes = episodes, nextCursor = page.nextCursor, loading = false, loadingMore = false, downloadedIds = downloadedIds(episodes))
                    if (cursor == null) reconcileDeviceDownloads(show, page.items)
                }
                .onFailure { failure ->
                    val current = mutableState.value
                    if (!current.matchesEpisodeRequest(request, show)) return@onFailure
                    if (cursor != null && (failure as? PodcastClientException)?.status == 409) {
                        mutableState.value = current.copy(episodes = emptyList(), nextCursor = null, loading = true, loadingMore = false, error = null)
                        loadEpisodes(null)
                    } else fail("Could not load episodes for this show.")
                }
        }
    }

    private fun loadDownloadedEpisodes(show: PodcastShow) {
        val request = mutableState.value
        viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { downloads.resolveDownloaded(request.libraryId, show, request.episodeSort) }
            val current = mutableState.value
            if (!current.matchesEpisodeRequest(request, show)) return@launch
            PerformanceDiagnostics.record("network", result.diagnostic)
            mutableState.value = current.copy(
                episodes = result.episodes, nextCursor = null, loading = false, loadingMore = false, error = null,
                downloadedIds = result.downloadedIds,
                message = downloadedEpisodeMessage(result) ?: current.message,
                messageIsError = false,
            )
        }
    }

    private fun reconcileDeviceDownloads(show: PodcastShow, newest: List<AudioTrack>) {
        val libraryId = mutableState.value.libraryId
        viewModelScope.launch {
            val result = withContext(Dispatchers.IO) { downloads.applyAutomaticPolicy(libraryId, show, newest) } ?: return@launch
            val ids = withContext(Dispatchers.IO) { downloads.ids() }
            mutableState.value = mutableState.value.copy(
                downloadedIds = ids,
                message = result.exceptionOrNull()?.let { "Automatic episode download could not finish." },
                messageIsError = result.isFailure,
            )
        }
    }

    // downloads.ids() ultimately touches disk (file stats, and for externally-stored episodes a real
    // Binder IPC to the storage provider via DownloadLocationStore) -- never call it off Dispatchers.IO,
    // even indirectly through the MutableStateFlow initializer or a synchronous PodcastState rebuild.
    private fun refreshDownloadedIds() {
        viewModelScope.launch {
            val ids = withContext(io) { downloads.ids() }
            mutableState.value = mutableState.value.copy(downloadedIds = ids)
        }
    }

    private fun fail(message: String) { mutableState.value = mutableState.value.copy(loading = false, loadingMore = false, error = message) }
    private fun downloadedIds(episodes: List<AudioTrack>): Set<String> =
        downloads.downloadedIds(episodes)

    private fun AudioTrack.key() = identityKey
}

internal fun downloadedEpisodeMessage(result: PodcastDownloadedEpisodes): String? = when {
    result.matched > 0 -> "Reconnected ${result.matched} downloaded episode${if (result.matched == 1) "" else "s"}."
    result.serverDownloadCount > 0 -> "Found ${result.serverDownloadCount} episode${if (result.serverDownloadCount == 1) "" else "s"} already stored on the server."
    result.episodes.isEmpty() && result.recoveredCount > 0 -> "Recovered podcast files were found, but none matched this RSS show."
    result.episodes.isEmpty() && result.unclassifiedCount > 0 -> "Recovered audio files were found, but none matched this RSS show's episode metadata."
    else -> null
}

internal fun PodcastState.matchesEpisodeRequest(request: PodcastState, show: PodcastShow): Boolean =
    libraryId == request.libraryId && selectedShow?.id == show.id &&
        playedFilter == request.playedFilter && episodeSort == request.episodeSort

internal fun subscriptionFailure(failure: Throwable): String = when ((failure as? PodcastClientException)?.status) {
    400 -> failure.message?.lineSequence()?.firstOrNull()?.take(180) ?: "That podcast feed could not be added."
    404 -> "That podcast subscription no longer exists."
    else -> "Could not update podcast subscriptions. Check the connection and try again."
}
