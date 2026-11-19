package net.kdhmedia.mochil.audio

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.catalogue.BrowseLayoutStore
import net.kdhmedia.mochil.playback.PlaybackQualityPreferences

class MusicViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = AudioClient()
    private val catalogue = credential?.let { MusicCatalogueRepository(it, client) }
    private val downloads = OfflineAudioStore(application)
    private val downloadCoordinator = MusicDownloadCoordinator(downloads, client::streamUrl)
    private val playbackQuality = PlaybackQualityPreferences(application)
    private val preferences = application.getSharedPreferences("music_view", 0)
    private val contentRequests = MusicCatalogueRequests(viewModelScope)
    private val libraryVisits = MusicRequestGeneration()
    private var batchPlanning = false
    private val emptyAudiobookRetries = mutableSetOf<String>()
    private val layouts = BrowseLayoutStore(application)
    private val mutableState = MutableStateFlow(MusicState(
        grouping = runCatching { MusicGrouping.valueOf(preferences.getString("grouping", MusicGrouping.Discover.name)!!) }.getOrDefault(MusicGrouping.Discover),
        layout = if (layouts.isList()) MusicLayout.List else MusicLayout.Covers,
    ))
    private val layoutListener = layouts.listen { list -> mutableState.value = mutableState.value.copy(layout = if (list) MusicLayout.List else MusicLayout.Covers) }
    val state: StateFlow<MusicState> = mutableState.asStateFlow()

    private val favorites by lazy {
        catalogue?.let { repository -> MusicFavoriteRequests(
            scope = viewModelScope, read = repository::favorites, write = repository::setFavorite,
            current = { mutableState.value }, publish = { mutableState.value = it }, notify = ::notifyAction,
        ) }
    }

    private val offlineRefresh = OfflineAudioRefresh(
        scope = viewModelScope,
        load = { libraryId -> withContext(Dispatchers.IO) { downloadCoordinator.snapshot(libraryId) } },
        onFailure = { libraryId, _ ->
            val current = mutableState.value
            if (current.libraryId.takeIf(String::isNotBlank) == libraryId) {
                mutableState.value = current.copy(downloadMessage = "Could not check offline downloads. Check storage access and try again.")
            }
        },
        publish = { libraryId, offline ->
            val current = mutableState.value
            if (current.libraryId.takeIf(String::isNotBlank) == libraryId) {
                mutableState.value = current.copy(downloadedIds = offline.ids, offlineTracks = offline.tracks)
            }
        },
    )

    init { refreshOfflineSnapshot(null) }

    fun open(libraryId: String, libraryKind: String = "music") {
        if (libraryId == mutableState.value.libraryId && !shouldReloadAudioLibrary(mutableState.value, libraryKind)) return
        contentRequests.advance()
        libraryVisits.advance()
        val current = mutableState.value
        val grouping = initialAudioGrouping(libraryKind, current.grouping)
        mutableState.value = MusicState(
            libraryId = libraryId, libraryKind = libraryKind, grouping = grouping,
            layout = current.layout, loading = true, downloadedIds = current.downloadedIds,
            downloadedOnly = current.downloadedOnly, favoriteIds = current.favoriteIds,
            favoriteTracks = current.favoriteTracks,
        )
        refreshOfflineSnapshot(libraryId)
        loadInitial()
        loadFavorites()
    }

    fun setGrouping(grouping: MusicGrouping) {
        val current = mutableState.value
        if (current.grouping == grouping) return
        contentRequests.advance()
        preferences.edit().putString("grouping", grouping.name).apply()
        mutableState.value = current.copy(
            grouping = grouping, selectedGroup = null, selectedArtist = null, groups = emptyList(),
            artistAlbums = emptyList(), tracks = emptyList(), query = "", nextCursor = null,
            nextOffset = null, downloadedOnly = false, favoritesOnly = grouping == MusicGrouping.Favorites,
            loading = true, loadingMore = false, error = null,
            navigation = emptyList(), bucket = "",
        )
        loadInitial()
        if (grouping == MusicGrouping.Favorites) loadFavorites()
    }

    fun setLayout(layout: MusicLayout) {
        layouts.setList(layout == MusicLayout.List)
        mutableState.value = mutableState.value.copy(layout = layout)
    }

    fun refreshOfflineDownloads() {
        refreshOfflineSnapshot(mutableState.value.libraryId.takeIf(String::isNotBlank))
    }

    fun search(query: String) {
        val current = mutableState.value
        val normalized = query.trim()
        if (current.query == normalized) return
        contentRequests.advance()
        mutableState.value = current.copy(query = normalized, groups = emptyList(), tracks = emptyList(), nextCursor = null, nextOffset = null, navigation = emptyList(), bucket = "", loading = true, error = null)
        loadInitial()
    }

    fun openGroup(group: AudioGroup) {
        val current = mutableState.value
        if (current.grouping == MusicGrouping.Artists && current.selectedArtist == null) {
            loadArtistAlbums(group)
            return
        }
        openAlbum(group)
    }

    fun openAlbum(group: AudioGroup) {
        val current = mutableState.value
        val generation = contentRequests.advance()
        mutableState.value = current.copy(selectedGroup = group, tracks = emptyList(), loading = true, error = null)
        val repository = catalogue ?: return showUnavailableMusic()
        contentRequests.launch {
            val grouping = if (current.selectedArtist != null) MusicGrouping.Albums else current.grouping
            runCatching { repository.groupTracks(current.libraryId, grouping, group, current.selectedArtist) }
                .onSuccess { tracks -> if (contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(tracks = tracks, loading = false) }
                .onFailure { if (it !is CancellationException && contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(loading = false, error = "Could not open this music group.") }
        }
    }

    fun closeGroup() {
        val current = mutableState.value
        val parent = current.withClosedMusicGroup()
        if (parent === current) return
        contentRequests.advance()
        mutableState.value = parent
    }

    fun closeArtist() {
        val current = mutableState.value
        val parent = current.withClosedMusicArtist()
        if (parent === current) return
        contentRequests.advance()
        mutableState.value = parent
    }

    fun retry() {
        val current = mutableState.value
        current.selectedGroup?.let { openAlbum(it); return }
        current.selectedArtist?.let { loadArtistAlbums(it); return }
        if (current.tracks.isEmpty()) {
            mutableState.value = current.copy(loading = true, error = null)
            loadInitial()
        } else loadMore()
    }

    fun loadMore() {
        val current = mutableState.value
        if (!current.canLoadMoreMusic()) return
        if (!current.grouping.isPagedTrackView()) {
            val offset = current.nextOffset ?: return
            mutableState.value = current.copy(loadingMore = true, error = null)
            loadGroups(offset)
            return
        }
        val cursor = current.nextCursor ?: return
        mutableState.value = current.copy(loadingMore = true, error = null)
        load(current.libraryId, cursor)
    }

    fun jumpToBucket(bucket: String) {
        val current = mutableState.value
        val jump = current.musicBucketJump(bucket) ?: return
        contentRequests.advance()
        mutableState.value = jump.state
        if (current.grouping.isPagedTrackView()) load(current.libraryId, null, jump.offset)
        else loadGroups(jump.offset)
    }

    fun streamUrl(track: AudioTrack) = credential?.let { client.streamUrl(it, track) }.orEmpty()
    // downloadCoordinator.playbackUrl() checks disk and, for externally-stored tracks, makes a real
    // Binder IPC via DownloadLocationStore to the storage provider -- never call it off Dispatchers.IO.
    suspend fun playbackUrl(track: AudioTrack): String =
        withContext(Dispatchers.IO) { downloadCoordinator.playbackUrl(track) } ?: credential?.let { client.playbackUrl(it, track, playbackQuality.effective()) }.orEmpty()
    fun artworkUrl(track: AudioTrack) = credential?.let { client.artworkUrl(it, track) }.orEmpty()
    fun artworkUrl(group: AudioGroup) = credential?.let { client.artworkUrl(it, group) }.orEmpty()

    fun setDownloadedOnly(enabled: Boolean) {
        mutableState.value = mutableState.value.copy(
            downloadedOnly = enabled,
            favoritesOnly = false,
            error = if (enabled) null else mutableState.value.error,
            loading = false,
        )
    }

    fun setFavoritesOnly(enabled: Boolean) {
        mutableState.value = mutableState.value.copy(favoritesOnly = enabled, downloadedOnly = false, loading = false, error = null)
    }

    fun showAllTracks() {
        mutableState.value = mutableState.value.copy(downloadedOnly = false, favoritesOnly = false)
    }

    fun toggleFavorite(track: AudioTrack) {
        favorites?.toggle(track)
    }

    fun toggleDownload(track: AudioTrack) {
        val key = track.identityKey
        val current = mutableState.value
        val libraryId = current.libraryId
        val visit = libraryVisits.capture()
        if (current.downloadingId != null) return
        mutableState.value = current.copy(downloadingId = key, downloadMessage = null)
        viewModelScope.launch {
            val result = withContext(Dispatchers.IO) {
                runCatching {
                    downloadCoordinator.toggle(credential, track)
                }
            }
            refreshOfflineDownloads()
            if (!libraryVisits.isCurrent(visit) || mutableState.value.libraryId != libraryId) return@launch
            mutableState.value = mutableState.value.copy(
                downloadingId = null,
                downloadMessage = result.exceptionOrNull()?.message,
            )
            if (result.isSuccess) notifyAction(if (result.getOrDefault(false)) "Available offline." else "Removed offline download.")
        }
    }

    fun notifyAction(message: String) {
        mutableState.value = mutableState.value.copy(actionMessage = message)
        viewModelScope.launch {
            delay(2_600)
            if (mutableState.value.actionMessage == message) mutableState.value = mutableState.value.copy(actionMessage = null)
        }
    }

    fun downloadSelectedGroup() {
        downloadTracks(mutableState.value.tracks, "group")
    }

    fun downloadTracks(tracks: List<AudioTrack>, label: String = "playlist") {
        val current = mutableState.value
        val libraryId = current.libraryId
        val visit = libraryVisits.capture()
        if (current.batchDownloadTotal > 0 || batchPlanning) return
        val saved = credential ?: return
        batchPlanning = true
        viewModelScope.launch {
            val pending = prepareDownloadBatch(tracks, false, visit) ?: return@launch
            if (pending.isEmpty()) {
                notifyAction("This $label is already available offline.")
                return@launch
            }
            mutableState.value = mutableState.value.copy(batchDownloadCompleted = 0, batchDownloadTotal = pending.size, batchDownloadLabel = label, downloadMessage = null)
            val result = withContext(Dispatchers.IO) { downloadCoordinator.downloadBatch(saved, pending) { completed, track ->
                withContext(Dispatchers.Main.immediate) {
                    if (libraryVisits.isCurrent(visit)) {
                        val latest = mutableState.value
                        mutableState.value = latest.copy(batchDownloadCompleted = completed, downloadedIds = latest.downloadedIds + track.downloadKey)
                    }
                }
            } }
            refreshOfflineDownloads()
            if (!libraryVisits.isCurrent(visit) || mutableState.value.libraryId != libraryId) return@launch
            mutableState.value = mutableState.value.copy(
                batchDownloadCompleted = 0,
                batchDownloadTotal = 0,
                batchDownloadLabel = "",
                downloadMessage = result.failure?.message,
            )
            if (result.failure == null) notifyAction("Downloaded ${result.completed} ${if (result.completed == 1) "track" else "tracks"}.")
        }
    }

    fun removeDownloads(tracks: List<AudioTrack>, label: String = "playlist") {
        val current = mutableState.value
        val libraryId = current.libraryId
        val visit = libraryVisits.capture()
        if (current.batchDownloadTotal > 0 || batchPlanning) return
        batchPlanning = true
        viewModelScope.launch {
            val downloaded = prepareDownloadBatch(tracks, true, visit) ?: return@launch
            if (downloaded.isEmpty()) {
                notifyAction("This $label has no offline downloads.")
                return@launch
            }
            mutableState.value = mutableState.value.copy(batchDownloadCompleted = 0, batchDownloadTotal = downloaded.size, batchDownloadLabel = label, downloadMessage = null)
            val result = withContext(Dispatchers.IO) { downloadCoordinator.removeBatch(downloaded) { completed, _ ->
                withContext(Dispatchers.Main.immediate) {
                    if (libraryVisits.isCurrent(visit)) {
                        mutableState.value = mutableState.value.copy(batchDownloadCompleted = completed)
                    }
                }
            } }
            refreshOfflineDownloads()
            if (!libraryVisits.isCurrent(visit) || mutableState.value.libraryId != libraryId) return@launch
            mutableState.value = mutableState.value.copy(
                batchDownloadCompleted = 0,
                batchDownloadTotal = 0,
                batchDownloadLabel = "",
                downloadMessage = result.failure?.message,
            )
            if (result.failure == null) notifyAction("Removed ${result.completed} offline ${if (result.completed == 1) "track" else "tracks"}.")
        }
    }

    private suspend fun prepareDownloadBatch(tracks: List<AudioTrack>, removing: Boolean, visit: Long): List<AudioTrack>? {
        return try {
            val selected = if (removing) downloadCoordinator.downloaded(tracks) else downloadCoordinator.pending(tracks)
            selected.takeIf { libraryVisits.isCurrent(visit) }
        } catch (cancelled: kotlinx.coroutines.CancellationException) {
            throw cancelled
        } catch (_: Exception) {
            if (libraryVisits.isCurrent(visit)) {
                mutableState.value = mutableState.value.copy(downloadMessage = "Could not check offline downloads. Check storage access and try again.")
            }
            null
        } finally {
            batchPlanning = false
        }
    }

    fun artistQueue(shuffle: Boolean, onReady: (List<AudioTrack>) -> Unit) {
        val current = mutableState.value
        val artist = current.selectedArtist ?: return
        val repository = catalogue ?: return
        val generation = contentRequests.capture()
        contentRequests.launch {
            runCatching { repository.artistTracks(current.libraryId, artist) }
                .onSuccess { tracks -> if (contentRequests.isCurrent(generation)) onReady(if (shuffle) tracks.shuffled() else tracks) }
                .onFailure { if (it !is CancellationException && contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(error = "Could not prepare this artist.") }
        }
    }

    private fun loadArtistAlbums(artist: AudioGroup) {
        val current = mutableState.value
        val generation = contentRequests.advance()
        mutableState.value = current.copy(selectedArtist = artist, artistAlbums = emptyList(), loading = true, error = null)
        val repository = catalogue ?: return showUnavailableMusic()
        contentRequests.launch {
            runCatching { repository.artistAlbums(current.libraryId, artist) }
                .onSuccess { albums -> if (contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(artistAlbums = albums, loading = false) }
                .onFailure { if (it !is CancellationException && contentRequests.isCurrent(generation)) mutableState.value = mutableState.value.copy(loading = false, error = "Could not load this artist's albums.") }
        }
    }

    private fun loadFavorites() {
        favorites?.refresh()
    }

    private fun refreshOfflineSnapshot(libraryId: String?) {
        offlineRefresh.refresh(libraryId)
    }

    private fun loadInitial() {
        val current = mutableState.value
        if (current.libraryKind == "music" && current.grouping in setOf(MusicGrouping.Albums, MusicGrouping.Artists, MusicGrouping.Tracks)) loadNavigation(current)
        when (current.grouping) {
            MusicGrouping.Discover -> loadDiscovery()
            MusicGrouping.RecentlyPlayed -> loadRecentlyPlayed()
            MusicGrouping.Favorites -> mutableState.value = current.copy(loading = false, error = null, favoritesOnly = true)
            MusicGrouping.Playlists -> mutableState.value = current.copy(loading = false, error = null)
            MusicGrouping.Continue, MusicGrouping.Recent, MusicGrouping.Tracks -> load(current.libraryId, null)
            else -> loadGroups(0)
        }
    }

    private fun loadDiscovery() {
        val repository = catalogue ?: return showUnavailableMusic()
        val request = mutableState.value
        val generation = contentRequests.capture()
        contentRequests.launch {
            val result = runCatching { repository.discovery(request.libraryId) }
            result.onSuccess { (albums, artists, recent, played) ->
                val current = mutableState.value
                if (!contentRequests.isCurrent(generation) || current.libraryId != request.libraryId || current.grouping != MusicGrouping.Discover) return@onSuccess
                mutableState.value = current.copy(discoveryAlbums = albums, discoveryArtists = artists, discoveryRecent = recent, discoveryPlayed = played, loading = false, error = null, offlineOnly = false)
            }.onFailure { failure -> handleFailure(generation, failure, "Could not load music discovery.", true) }
        }
    }

    private fun loadRecentlyPlayed() {
        val repository = catalogue ?: return showUnavailableMusic()
        val request = mutableState.value
        val generation = contentRequests.capture()
        contentRequests.launch {
            runCatching { repository.recentlyPlayed(request.libraryId) }
                .onSuccess { tracks ->
                    val current = mutableState.value
                    if (!contentRequests.isCurrent(generation) || current.libraryId != request.libraryId || current.grouping != MusicGrouping.RecentlyPlayed) return@onSuccess
                    mutableState.value = current.copy(discoveryPlayed = tracks, loading = false, error = null)
                }
                .onFailure { failure -> handleFailure(generation, failure, "Could not load recently played music.", true) }
        }
    }

    private fun loadGroups(offset: Int) {
        val repository = catalogue ?: return showUnavailableMusic()
        val request = mutableState.value
        val generation = contentRequests.capture()
        contentRequests.launch {
            runCatching { repository.groups(request.libraryId, request.grouping, offset, request.query) }
                .onSuccess { page ->
                    val current = mutableState.value
                    if (!contentRequests.isCurrent(generation) || current.libraryId != request.libraryId || current.grouping != request.grouping || current.query != request.query) return@onSuccess
                    val emptyAudiobookAction = emptyAudiobookPageAction(request, offset, page.items, emptyAudiobookRetries)
                    if (emptyAudiobookAction != EmptyAudiobookPageAction.None) {
                        if (emptyAudiobookAction == EmptyAudiobookPageAction.Retry) delay(1_000)
                        if (contentRequests.isCurrent(generation)) {
                            if (emptyAudiobookAction == EmptyAudiobookPageAction.Retry) loadGroups(0)
                            else setGrouping(MusicGrouping.Tracks)
                        }
                        return@onSuccess
                    }
                    if (page.items.isNotEmpty()) emptyAudiobookRetries.remove(request.libraryId)
                    mutableState.value = current.copy(
                        groups = if (offset == 0) page.items else current.groups + page.items,
                        loading = false, loadingMore = false, nextOffset = page.nextOffset, error = null, offlineOnly = false,
                    )
                }
                .onFailure { failure -> handleFailure(generation, failure, "Could not load music groups.", offset == 0) }
        }
    }

    private fun load(libraryId: String, cursor: String?, offset: Int = 0) {
        val repository = catalogue ?: return showUnavailableMusic()
        val requestQuery = mutableState.value.query
        val requestGrouping = mutableState.value.grouping
        val generation = contentRequests.capture()
        contentRequests.launch {
            val sort = when (requestGrouping) {
                MusicGrouping.Tracks -> "title"
                MusicGrouping.Continue, MusicGrouping.Recent -> "recent"
                else -> ""
            }
            val played = if (requestGrouping == MusicGrouping.Continue) "continue" else ""
            runCatching { repository.tracks(libraryId, cursor, requestQuery, sort, played, offset) }
                .onSuccess { page ->
                    val current = mutableState.value
                    if (!contentRequests.isCurrent(generation) || current.libraryId != libraryId || current.query != requestQuery || current.grouping != requestGrouping) return@onSuccess
                    mutableState.value = current.copy(
                        tracks = if (cursor == null) page.items else current.tracks + page.items,
                        loading = false,
                        loadingMore = false,
                        nextCursor = page.nextCursor,
                        error = null,
                        offlineOnly = false,
                    )
                }
                .onFailure { failure ->
                    if (mutableState.value.libraryId == libraryId) handleFailure(generation, failure, "Could not load this music shelf.", cursor == null)
                }
        }
    }

    private fun loadNavigation(request: MusicState) {
        val repository = catalogue ?: return
        val generation = contentRequests.capture()
        contentRequests.launch {
            runCatching { repository.navigation(request.libraryId, request.grouping, request.query) }
                .onSuccess { navigation ->
                    val current = mutableState.value
                    if (contentRequests.isCurrent(generation) && current.libraryId == request.libraryId && current.grouping == request.grouping && current.query == request.query) {
                        mutableState.value = current.copy(navigation = navigation)
                    }
                }
        }
    }

    private fun handleFailure(generation: Long, failure: Throwable, message: String, initial: Boolean) {
        if (failure is CancellationException || !contentRequests.isCurrent(generation)) return
        if ((failure as? AudioClientException)?.status == 401) {
            net.kdhmedia.mochil.auth.DeviceSessionEvents.expired()
            mutableState.value = mutableState.value.copy(loading = false, loadingMore = false, error = "This device connection expired. Sign in again.")
            return
        }
        val current = mutableState.value
        mutableState.value = if (initial) current.offlineFallback() ?: current.copy(loading = false, loadingMore = false, error = message)
        else current.copy(loading = false, loadingMore = false, error = message)
    }

    private fun showUnavailableMusic() {
        val current = mutableState.value
        mutableState.value = current.offlineFallback()
            ?: current.copy(loading = false, loadingMore = false, error = "Connect a server to browse this music shelf.")
    }
}

internal enum class EmptyAudiobookPageAction { None, Retry, ShowAll }

internal fun emptyAudiobookPageAction(
    request: MusicState,
    offset: Int,
    items: List<AudioGroup>,
    retriedLibraries: MutableSet<String>,
): EmptyAudiobookPageAction {
    if (request.libraryKind != "audiobooks" || offset != 0 || request.query.isNotBlank() || items.isNotEmpty()) {
        return EmptyAudiobookPageAction.None
    }
    return if (retriedLibraries.add(request.libraryId)) EmptyAudiobookPageAction.Retry else EmptyAudiobookPageAction.ShowAll
}

internal fun initialAudioGrouping(libraryKind: String, current: MusicGrouping): MusicGrouping =
    if (libraryKind == "audiobooks") {
        MusicGrouping.Series
    } else {
        current.takeIf {
            it == MusicGrouping.Discover ||
                it == MusicGrouping.RecentlyPlayed ||
                it == MusicGrouping.Albums ||
                it == MusicGrouping.Artists ||
                it == MusicGrouping.Favorites ||
                it == MusicGrouping.Playlists ||
                it == MusicGrouping.Tracks
        } ?: MusicGrouping.Discover
    }

internal fun shouldReloadAudioLibrary(state: MusicState, requestedKind: String): Boolean =
    state.libraryKind != requestedKind ||
        state.error != null ||
        !state.loading && state.groups.isEmpty() && state.tracks.isEmpty() && state.discoveryAlbums.isEmpty() &&
        state.discoveryArtists.isEmpty() && state.discoveryRecent.isEmpty() && state.discoveryPlayed.isEmpty()
