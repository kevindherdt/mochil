package net.kdhmedia.mochil.video

import android.app.Application
import android.content.Intent
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.playback.PlaybackQuality
import net.kdhmedia.mochil.playback.PlaybackQualityPreferences

class VideoViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = VideoClient()
    private val preferences = VideoBrowsePreferences(application)
    private val playbackQuality = PlaybackQualityPreferences(application)
    private var loadJob: Job? = null
    private var pageJob: Job? = null
    private val contentRequests = VideoRequestGeneration()
    private val searchDebounce = VideoSearchDebounce(viewModelScope, contentRequests)
    private var pageGeneration = 0L
    private var detailsLibraryGeneration = 0L
    private val mutableState = MutableStateFlow(VideoState())
    val state = mutableState.asStateFlow()
    private val layoutListener = preferences.listen { layout -> mutableState.value = mutableState.value.copy(layout = layout) }

    fun open(libraryId: String, kind: String) {
        if (mutableState.value.libraryId == libraryId && mutableState.value.libraryKind == kind) return
        detailsLibraryGeneration++
        mutableState.value = VideoState(libraryId = libraryId, libraryKind = kind, layout = preferences.layout(kind), loading = true)
        loadRoot()
    }

    fun openShow(show: VideoShow, season: Int? = show.seasons.firstOrNull()) {
        mutableState.value = mutableState.value.copy(selectedShow = show, selectedSeason = season, loading = true, error = null, items = emptyList())
        loadItems(show, season)
    }

    fun selectSeason(season: Int) = mutableState.value.selectedShow?.let { openShow(it, season) }
    fun closeShow() { mutableState.value = mutableState.value.copy(selectedShow = null, selectedSeason = null, items = emptyList()); loadRoot() }
    fun retry() {
        if (mutableState.value.pagingError != null) loadMore()
        else if (mutableState.value.selectedShow == null) loadRoot()
        else loadItems(mutableState.value.selectedShow, mutableState.value.selectedSeason)
    }
    fun refreshAfterPlayback() {
        val current = mutableState.value
        if (current.libraryId.isBlank()) return
        if (current.selectedShow == null) loadRoot()
        else {
            mutableState.value = current.copy(loading = true, error = null)
            loadItems(current.selectedShow, current.selectedSeason)
        }
    }

    fun setLayout(layout: VideoLayout) {
        val current = mutableState.value
        preferences.setLayout(current.libraryKind, layout)
        mutableState.value = current.copy(layout = layout)
    }

    fun setSearchOpen(open: Boolean) {
        mutableState.value = mutableState.value.copy(searchOpen = open)
        if (!open && mutableState.value.query.isNotBlank()) search("")
    }

    fun search(query: String) {
        loadJob?.cancel()
        pageJob?.cancel()
        mutableState.value = mutableState.value.copy(
            query = query, loading = true, error = null,
            // A new query replaces the list rather than filtering it in place: the previous query's rows must not sit
            // under the new one while it loads (the screen keeps content visible during a background refresh, so the
            // emptiness here is what makes a searched list still show its spinner and its error).
            shows = emptyList(), items = emptyList(),
            showLoadingMore = false, showNextOffset = null, showPagingError = null,
            itemLoadingMore = false, itemNextOffset = null, itemPagingError = null,
        )
        searchDebounce.submit {
            if (mutableState.value.selectedShow == null) loadRoot() else loadItems(mutableState.value.selectedShow, mutableState.value.selectedSeason)
        }
    }

    fun play(item: VideoItem) {
        val saved = credential ?: return
        val quality = playbackQuality.effective()
        val playbackPath = if (quality == PlaybackQuality.Maximum) item.streamPath.ifBlank { item.hlsPath } else client.playbackPath(item.hlsPath, quality)
        if (playbackPath.isBlank()) return
        val (window, currentIndex) = videoPlaybackWindow(mutableState.value.items, item)
        val queue = VideoPlaybackQueue(window.map { value ->
            VideoPlaybackEntry(
                libraryId = value.libraryId,
                itemId = value.id,
                title = value.episodeTitle.ifBlank { value.title }, revision = value.revision,
                url = client.absolute(saved, if (quality == PlaybackQuality.Maximum) value.streamPath.ifBlank { value.hlsPath } else client.playbackPath(value.hlsPath, quality)),
                fallbackUrl = if (quality == PlaybackQuality.Maximum && value.streamPath.isNotBlank() && value.hlsPath.isNotBlank()) client.absolute(saved, value.hlsPath) else "",
                positionMs = (value.resumeSeconds * 1_000).toLong().coerceAtLeast(0),
                durationSeconds = value.durationSeconds,
                subtitles = value.encodedSubtitles(),
                progressRevision = value.progressRevision,
                finished = value.finished,
            )
        }, currentIndex)
        val intent = Intent(getApplication(), VideoPlayerActivity::class.java).apply {
            putExtra(VideoPlayerActivity.EXTRA_LIBRARY_ID, item.libraryId)
            putExtra(VideoPlayerActivity.EXTRA_ITEM_ID, item.id)
            putExtra(VideoPlayerActivity.EXTRA_TITLE, item.displayTitle)
            putExtra(VideoPlayerActivity.EXTRA_URL, client.absolute(saved, playbackPath))
            if (quality == PlaybackQuality.Maximum && item.streamPath.isNotBlank() && item.hlsPath.isNotBlank()) putExtra(VideoPlayerActivity.EXTRA_FALLBACK_URL, client.absolute(saved, item.hlsPath))
            putExtra(VideoPlayerActivity.EXTRA_POSITION_MS, (item.resumeSeconds * 1_000).toLong())
            putExtra(VideoPlayerActivity.EXTRA_REVISION, item.revision)
            putExtra(VideoPlayerActivity.EXTRA_PROGRESS_REVISION, item.progressRevision)
            putExtra(VideoPlayerActivity.EXTRA_FINISHED, item.finished)
            putExtra(VideoPlayerActivity.EXTRA_DURATION_SECONDS, item.durationSeconds)
            putExtra(VideoPlayerActivity.EXTRA_SUBTITLES, item.encodedSubtitles())
            putExtra(VideoPlayerActivity.EXTRA_QUEUE, encodeVideoPlaybackQueue(queue))
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        getApplication<Application>().startActivity(intent)
    }

    private fun VideoItem.encodedSubtitles(): String = subtitles.joinToString("\n") {
        "${it.index}\t${it.language.replace(Regex("[\\t\\r\\n]"), " ")}\t${it.title.replace(Regex("[\\t\\r\\n]"), " ")}\t${it.default}"
    }

    fun dismissContinueWatching(item: VideoItem) {
        val saved = credential ?: return
        val generation = contentRequests.snapshot()
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.dismissContinueWatching(saved, item) } }
                .onSuccess { mutableState.value = mutableState.value.copy(continueWatching = mutableState.value.continueWatching.filterNot { it.libraryId == item.libraryId && it.id == item.id }) }
                .onFailure { failure ->
                    contentRequests.onCurrentFailure(generation, failure) {
                        mutableState.value = mutableState.value.copy(error = "Could not remove this video from Continue watching.")
                    }
                }
        }
    }

    fun setWatched(item: VideoItem, watched: Boolean) {
        val saved = credential ?: return
        val generation = contentRequests.snapshot()
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.setWatched(saved, item, watched) } }
                .onSuccess {
                    mutableState.value = mutableState.value.withConfirmedWatched(item, watched)
                }
                .onFailure { failure ->
                    contentRequests.onCurrentFailure(generation, failure) {
                        mutableState.value = mutableState.value.copy(error = "Could not mark this episode as ${if (watched) "watched" else "unwatched"}.")
                    }
                }
        }
    }

    /** Asks the server for a show's or movie's TMDB facts once; the page shows them when they arrive and nothing changes if there are none. */
    fun loadDetails(show: VideoShow?, item: VideoItem?) {
        val saved = credential ?: return
        val libraryId = mutableState.value.libraryId
        val key = if (show != null) "show:${show.sourceName}" else "item:${item?.id.orEmpty()}"
        if (libraryId.isBlank() || key in mutableState.value.details || (show == null && item == null)) return
        val generation = detailsLibraryGeneration
        viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.details(saved, libraryId, show?.sourceName, item?.id) } }.onSuccess { found ->
                if (generation == detailsLibraryGeneration) mutableState.value = mutableState.value.copy(details = mutableState.value.details + (key to found))
            }
        }
    }

    fun artworkUrl(path: String): String = credential?.let { client.absolute(it, path) }.orEmpty()
    fun authorizationHeader(): String? = credential?.let { "Bearer ${it.token}" }

    fun loadMore() {
        val current = mutableState.value
        if (current.selectedShow == null && current.libraryKind in setOf("tv", "anime")) loadMoreShows() else loadMoreItems()
    }

    fun loadMoreShows() = loadMorePage(shows = true)
    fun loadMoreItems() = loadMorePage(shows = false)

    private fun loadMorePage(shows: Boolean) {
        val saved = credential ?: return
        val current = mutableState.value
        if (shows != (current.selectedShow == null && current.libraryKind in setOf("tv", "anime"))) return
        val offset = (if (shows) current.showNextOffset else current.itemNextOffset) ?: return
        if (current.loading || if (shows) current.showLoadingMore else current.itemLoadingMore) return
        val generation = pageGeneration
        val libraryId = current.libraryId
        val query = current.query
        val show = current.selectedShow
        val season = current.selectedSeason
        mutableState.value = if (shows) current.copy(showLoadingMore = true, showPagingError = null)
            else current.copy(itemLoadingMore = true, itemPagingError = null)
        pageJob = viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                if (shows) VideoNextPage(shows = client.shows(saved, libraryId, query, offset))
                else VideoNextPage(items = client.videos(saved, libraryId, show, season, query, offset))
            } }.onSuccess { page ->
                val latest = mutableState.value
                val latestOffset = if (shows) latest.showNextOffset else latest.itemNextOffset
                if (contentRequests.isCurrent(generation) && latestOffset == offset) {
                    mutableState.value = if (shows) latest.copy(
                        shows = latest.shows + page.shows.items,
                        showNextOffset = page.shows.nextOffset,
                        showLoadingMore = false,
                        showPagingError = null,
                    ) else latest.copy(
                        items = latest.items + page.items.items,
                        itemNextOffset = page.items.nextOffset,
                        itemLoadingMore = false,
                        itemPagingError = null,
                    )
                }
            }.onFailure { failure ->
                if (failure !is CancellationException && contentRequests.isCurrent(generation)) {
                    val latest = mutableState.value
                    val message = failure.message ?: "Could not load more videos."
                    mutableState.value = if (shows) latest.copy(showLoadingMore = false, showPagingError = message)
                        else latest.copy(itemLoadingMore = false, itemPagingError = message)
                }
            }
        }
    }

    private fun loadRoot() {
        val saved = credential ?: return fail("This device is no longer connected.")
        val current = mutableState.value
        val generation = contentRequests.advance().also { pageGeneration = it }
        pageJob?.cancel()
        mutableState.value = current.copy(loading = true, showLoadingMore = false, showNextOffset = null, showPagingError = null, itemLoadingMore = false, itemNextOffset = null, itemPagingError = null, error = null)
        loadJob?.cancel()
        loadJob = viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) {
                val root = if (current.libraryKind in setOf("tv", "anime")) VideoNextPage(shows = client.shows(saved, current.libraryId, current.query))
                else VideoNextPage(items = client.videos(saved, current.libraryId, query = current.query))
                currentCoroutineContext().ensureActive()
                val watching = if (current.query.isBlank()) runCatching { client.continueWatching(saved) }.getOrDefault(emptyList()).filter {
                    it.libraryId == current.libraryId && !it.dismissed && it.resumeSeconds > 0 && it.progressFraction < .95f && it.playable
                }.take(12) else emptyList()
                root.copy(watching = watching)
            } }.onSuccess { root ->
                if (contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(
                    shows = root.shows.items,
                    items = root.items.items,
                    continueWatching = root.watching,
                    showNextOffset = root.shows.nextOffset,
                    itemNextOffset = root.items.nextOffset,
                    loading = false,
                )
            }.onFailure { failure ->
                if (failure !is CancellationException && contentRequests.isCurrent(generation)) fail(failure.message ?: "Could not load this video library.")
            }
        }
    }

    private fun loadItems(show: VideoShow?, season: Int?) {
        val saved = credential ?: return fail("This device is no longer connected.")
        val libraryId = mutableState.value.libraryId
        val query = mutableState.value.query
        val generation = contentRequests.advance().also { pageGeneration = it }
        pageJob?.cancel()
        mutableState.value = mutableState.value.copy(itemLoadingMore = false, itemNextOffset = null, itemPagingError = null)
        loadJob?.cancel()
        loadJob = viewModelScope.launch {
            runCatching { withContext(Dispatchers.IO) { client.videos(saved, libraryId, show, season, query) } }
                .onSuccess { if (contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(items = it.items, itemNextOffset = it.nextOffset, loading = false) }
                .onFailure { failure ->
                    if (failure !is CancellationException && contentRequests.isCurrent(generation)) fail(failure.message ?: "Could not load these videos.")
                }
        }
    }

    private fun fail(message: String) { mutableState.value = mutableState.value.copy(loading = false, error = message) }
}

private data class VideoNextPage(
    val shows: VideoPage<VideoShow> = VideoPage(emptyList(), null),
    val items: VideoPage<VideoItem> = VideoPage(emptyList(), null),
    val watching: List<VideoItem> = emptyList(),
)

internal fun VideoItem.withWatched(watched: Boolean) = copy(
    resumeSeconds = if (watched) durationSeconds else 0.0,
    finished = watched,
    dismissed = false,
)

internal fun videoWatchedDelta(previous: Boolean, watched: Boolean): Int = when {
    watched && !previous -> 1
    !watched && previous -> -1
    else -> 0
}
