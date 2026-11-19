package net.kdhmedia.mochil.dashboard

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.catalogue.ReadingStatus
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.video.VideoClient
import net.kdhmedia.mochil.video.VideoItem

class DashboardViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = DashboardClient()
    private val videoClient = VideoClient()
    private val mutableState = MutableStateFlow(DashboardState())
    val state: StateFlow<DashboardState> = mutableState.asStateFlow()
    private var refreshGeneration = 0L
    private var refreshInFlight = false
    private var refreshPending = false
    private var mutationGeneration = 0L
    private var lastHeldGeneration = 0L

    init { refresh() }

    fun refresh() = refresh(showLoading = true)

    /**
     * A refresh behind content that is already on screen: the app returning to the foreground. It deliberately does
     * not raise the loading flag, because that flag is what shows the pull-to-refresh indicator (and the home
     * skeleton), so every return from the reader or the player ran the spinner as if the reader had pulled.
     */
    fun refreshQuietly() = refresh(showLoading = false)

    private fun refresh(showLoading: Boolean) {
        if (refreshInFlight) {
            refreshPending = true
            return
        }
        val saved = credential ?: return run { mutableState.value = DashboardState(loading = false, error = "This device is no longer connected.") }
        refreshInFlight = true
        val generation = ++refreshGeneration
        if (showLoading) mutableState.value = mutableState.value.copy(loading = true, error = null)
        viewModelScope.launch {
            val snapshot = runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) {
                PerformanceDiagnostics.measure("network", "load_dashboard_home") { client.home(saved) }
            } }.getOrNull()
            if (snapshot != null) {
                refreshInFlight = false
                val current = generation == refreshGeneration
                if (current) {
                    mutableState.value = snapshot.toDashboardState().copy(held = mutableState.value.held, finished = mutableState.value.finished)
                }
                if (current) loadHeld(saved, generation)
                if (refreshPending) { refreshPending = false; refresh(showLoading = !current); return@launch }
                return@launch
            }
            val reading = async(Dispatchers.IO) { runCatching { client.reading(saved) } }
            val listening = async(Dispatchers.IO) { runCatching { client.listening(saved) } }
            val watching = async(Dispatchers.IO) { runCatching { videoClient.continueWatching(saved) } }
            val readingResult = reading.await()
            val listeningResult = listening.await()
            val watchingResult = watching.await()
            refreshInFlight = false
            val current = generation == refreshGeneration
            if (current) mutableState.value = DashboardState(
                reading = readingResult.getOrNull()?.let(::homeItems) ?: mutableState.value.reading,
                listening = listeningResult.getOrNull()?.let { homeItems(it.filter(::isLongFormListening)) } ?: mutableState.value.listening,
                watching = watchingResult.getOrNull()?.let { homeItems(it.filter { item -> !item.dismissed && item.resumeSeconds > 0 && item.progressFraction < .95f && item.playable }) } ?: mutableState.value.watching,
                loading = false,
                held = mutableState.value.held,
                finished = mutableState.value.finished,
                error = if (readingResult.isFailure && listeningResult.isFailure && watchingResult.isFailure) "Could not load your activity." else null,
            )
            if (current) loadHeld(saved, generation)
            if (refreshPending) { refreshPending = false; refresh(showLoading = !current); return@launch }
        }
    }

    private fun loadHeld(saved: net.kdhmedia.mochil.auth.DeviceCredential, generation: Long) {
        val mutationAtStart = mutationGeneration
        viewModelScope.launch {
            val held = runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { client.held(saved) } }.getOrNull()
            val finished = runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { client.finished(saved) } }.getOrNull()
            if (mutationAtStart != mutationGeneration || generation < lastHeldGeneration) return@launch
            lastHeldGeneration = generation
            mutableState.value = mutableState.value.copy(held = held ?: mutableState.value.held, finished = finished ?: mutableState.value.finished)
        }
    }

    /** Puts a shelved or finished book back into Continue reading, keeping its place. */
    fun resumeHeld(book: BookSummary) {
        val saved = credential ?: return
        refreshGeneration++
        mutationGeneration++
        viewModelScope.launch {
            runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { net.kdhmedia.mochil.catalogue.ReadingStatusClient().set(saved, book.libraryId, book.id, ReadingStatus.Reading.wire) } }
                .onSuccess {
                    // Invalidate held/finished requests that started while the resume was in flight.
                    mutationGeneration++
                    mutableState.value = mutableState.value.copy(
                        held = mutableState.value.held.filterNot { it.book.libraryId == book.libraryId && it.book.id == book.id },
                        finished = mutableState.value.finished.filterNot { it.book.libraryId == book.libraryId && it.book.id == book.id },
                    )
                    refresh()
                }
                .onFailure { mutableState.value = mutableState.value.copy(error = "Could not resume this book.", loading = false) }
        }
    }

    fun dismissReading(book: BookSummary) {
        val saved = credential ?: return
        refreshGeneration++
        mutationGeneration++
        viewModelScope.launch {
            runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { client.dismissReading(saved, book) } }
                .onSuccess { mutableState.value = mutableState.value.copy(reading = mutableState.value.reading.filterNot { it.book.libraryId == book.libraryId && it.book.id == book.id }, loading = false) }
                .onFailure { mutableState.value = mutableState.value.copy(error = "Could not remove this book from Home.", loading = false) }
        }
    }

    fun dismissWatching(item: VideoItem) {
        val saved = credential ?: return
        refreshGeneration++
        mutationGeneration++
        viewModelScope.launch {
            runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { videoClient.dismissContinueWatching(saved, item) } }
                .onSuccess { mutableState.value = mutableState.value.copy(watching = mutableState.value.watching.filterNot { it.libraryId == item.libraryId && it.id == item.id }, loading = false) }
                .onFailure { mutableState.value = mutableState.value.copy(error = "Could not remove this video from Home.", loading = false) }
        }
    }

    fun dismissListening(item: ContinueListening) = setListeningDismissed(item, true)

    fun resolveListening(item: ContinueListening, onResolved: (AudioTrack) -> Unit) {
        val saved = credential ?: return onResolved(item.track)
        viewModelScope.launch {
            val latest = runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { client.latestListening(saved, item) } }.getOrNull()
            onResolved((latest ?: item).track)
        }
    }

    private fun setListeningDismissed(item: ContinueListening, dismissed: Boolean) {
        val saved = credential ?: return
        refreshGeneration++
        mutationGeneration++
        viewModelScope.launch {
            runCatching { kotlinx.coroutines.withContext(Dispatchers.IO) { client.setListeningDismissed(saved, item, dismissed) } }
                .onSuccess {
                    mutableState.value = mutableState.value.copy(listening = if (dismissed) {
                        mutableState.value.listening.filterNot { it.track.libraryId == item.track.libraryId && it.track.id == item.track.id }
                    } else homeItems(listOf(item) + mutableState.value.listening), loading = false)
                }
                .onFailure { mutableState.value = mutableState.value.copy(error = "Could not update this item on Home.", loading = false) }
        }
    }
}

private fun DashboardSnapshot.toDashboardState() = DashboardState(
    reading = homeItems(reading),
    listening = homeItems(listening.filter(::isLongFormListening)),
    watching = homeItems(watching.filter { !it.dismissed && it.resumeSeconds > 0 && it.progressFraction < .95f && it.playable }),
    loading = false,
)

internal const val HOME_SECTION_LIMIT = 5
internal fun <T> homeItems(values: List<T>): List<T> = values.take(HOME_SECTION_LIMIT)
internal fun isLongFormListening(item: ContinueListening): Boolean = item.track.libraryKind == "audiobooks" || item.track.libraryKind == "podcasts"
