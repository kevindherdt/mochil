package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

data class MusicPlaylistState(
    val playlists: List<MusicPlaylistOverview> = emptyList(),
    val selectedPlaylist: MusicPlaylist? = null,
    val detailLoading: Boolean = false,
    val loading: Boolean = true,
    val saving: Boolean = false,
    val error: String? = null,
    val selectedName: String? = null,
    val notice: String? = null,
)

/** UI-dispatcher request owner; serializes reconciliation without owning HTTP. */
internal class MusicPlaylistRequests(
    private val scope: CoroutineScope,
    private val loadDetail: suspend (String) -> MusicPlaylist,
    private val load: suspend () -> List<MusicPlaylistOverview>,
) {
    private val mutable = MutableStateFlow(MusicPlaylistState())
    val state = mutable.asStateFlow()
    private val operations = Mutex()
    private var readGeneration = 0L
    private var selectionGeneration = 0L
    private var detailGeneration = 0L
    private var pendingWrites = 0
    private var noticeGeneration = 0L

    fun select(name: String?) {
        selectionGeneration++
        requestDetail(name)
    }

    private fun requestDetail(name: String?) {
        val selection = ++detailGeneration
        mutable.value = state.value.copy(selectedName = name, selectedPlaylist = null, detailLoading = name != null, error = null)
        if (name == null) return
        scope.launch {
            operations.withLock {
                if (selection != detailGeneration) return@withLock
                try {
                    val detail = loadDetail(name)
                    currentCoroutineContext().ensureActive()
                    if (selection == detailGeneration) mutable.value = state.value.copy(selectedPlaylist = detail, detailLoading = false)
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (_: Exception) {
                    currentCoroutineContext().ensureActive()
                    if (selection == detailGeneration) mutable.value = state.value.copy(detailLoading = false, error = "Could not load this playlist.")
                }
            }
        }
    }
    fun error(message: String) { mutable.value = state.value.copy(error = message, loading = false) }

    fun refresh() {
        val request = ++readGeneration
        mutable.value = state.value.copy(loading = true, error = null)
        scope.launch {
            operations.withLock {
                if (request != readGeneration) return@withLock
                try {
                    val lists = load()
                    currentCoroutineContext().ensureActive()
                    if (request == readGeneration) publish(lists)
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (_: Exception) {
                    currentCoroutineContext().ensureActive()
                    if (request == readGeneration) error("Could not load playlists.")
                }
            }
        }
    }

    fun mutate(message: String, rename: Pair<String, String>? = null, action: suspend () -> Unit) {
        readGeneration++
        pendingWrites++
        val selection = selectionGeneration
        mutable.value = state.value.copy(saving = true, loading = false, error = null)
        scope.launch {
            try {
                operations.withLock {
                    var saved = false
                    try {
                        action()
                        currentCoroutineContext().ensureActive()
                        saved = true
                        if (rename != null && selection == selectionGeneration && state.value.selectedName.equals(rename.first, ignoreCase = true)) {
                            mutable.value = state.value.copy(selectedName = rename.second)
                        }
                        val lists = load()
                        currentCoroutineContext().ensureActive()
                        publish(lists)
                        notify(message)
                    } catch (cancelled: CancellationException) { throw cancelled }
                    catch (failure: Exception) {
                        currentCoroutineContext().ensureActive()
                        if (failure is MusicPlaylistConflict) {
                            requestDetail(state.value.selectedName)
                            error(failure.message.orEmpty())
                        } else error(if (saved) "Playlist saved, but could not reload playlists." else "Could not update playlists.")
                    }
                }
            } finally {
                pendingWrites--
                mutable.value = state.value.copy(saving = pendingWrites > 0)
            }
        }
    }

    private fun publish(lists: List<MusicPlaylistOverview>) {
        val selected = state.value.selectedName?.let { name -> lists.firstOrNull { it.name.equals(name, ignoreCase = true) }?.name }
        mutable.value = state.value.copy(playlists = lists, loading = false, selectedName = selected)
        requestDetail(selected)
    }

    private fun notify(message: String) {
        val request = ++noticeGeneration
        mutable.value = state.value.copy(notice = message, error = null)
        scope.launch {
            delay(2_600)
            if (request == noticeGeneration) mutable.value = state.value.copy(notice = null)
        }
    }
}
