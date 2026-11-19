package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

data class MusicQueueState(val tracks: List<AudioTrack> = emptyList(), val loaded: Boolean = false, val loading: Boolean = false, val saving: Boolean = false, val error: String? = null)

/** Called on the UI dispatcher; transport and scope are injected for race tests. */
internal class MusicQueueRequests(
    private val scope: CoroutineScope,
    private val connected: Boolean,
    private val load: suspend () -> List<AudioTrack>,
    private val save: suspend (List<AudioTrack>) -> Unit,
) {
    private val mutable = MutableStateFlow(MusicQueueState())
    val state = mutable.asStateFlow()
    private val writes = Mutex()
    private var generation = 0L
    private var unsavedChanges = false
    private val pendingEdits = mutableListOf<(List<AudioTrack>) -> List<AudioTrack>>()

    fun edit(transform: (List<AudioTrack>) -> List<AudioTrack>) {
        if (!connected) { refresh(); return }
        if (state.value.loaded) { persist(transform(state.value.tracks)); return }
        pendingEdits.add(transform)
        refresh()
    }

    fun refresh() {
        if (!connected) {
            mutable.value = MusicQueueState(error = "This device is no longer connected.")
            return
        }
        // A refresh must not replace an optimistic edit while it is being saved.
        if (state.value.saving || state.value.loading) return
        // Retrying after a failed PUT must resend the retained local snapshot,
        // not replace it with an older server queue (including a cleared queue).
        if (unsavedChanges) { persist(state.value.tracks); return }
        val request = ++generation
        mutable.value = state.value.copy(loading = true, error = null)
        scope.launch {
            try {
                val tracks = load().toList()
                currentCoroutineContext().ensureActive()
                if (request == generation) {
                    mutable.value = MusicQueueState(tracks = tracks, loaded = true)
                    if (pendingEdits.isNotEmpty()) {
                        val updated = pendingEdits.fold(tracks) { queue, edit -> edit(queue) }
                        persist(updated)
                    }
                }
            } catch (cancelled: CancellationException) { throw cancelled }
            catch (_: Exception) {
                currentCoroutineContext().ensureActive()
                if (request == generation) {
                    pendingEdits.clear()
                    mutable.value = state.value.copy(loading = false, error = "Could not load the queue.")
                }
            }
        }
    }

    fun persist(tracks: List<AudioTrack>) {
        if (!connected) return
        pendingEdits.clear()
        val snapshot = tracks.toList()
        val request = ++generation
        unsavedChanges = true
        mutable.value = state.value.copy(tracks = snapshot, loaded = true, loading = false, saving = true, error = null)
        scope.launch {
            writes.withLock {
                if (request != generation) return@withLock
                try {
                    save(snapshot)
                    currentCoroutineContext().ensureActive()
                    if (request == generation) {
                        unsavedChanges = false
                        mutable.value = state.value.copy(saving = false)
                    }
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (_: Exception) {
                    currentCoroutineContext().ensureActive()
                    if (request == generation) mutable.value = state.value.copy(saving = false, error = "Could not save the queue.")
                }
            }
        }
    }
}
