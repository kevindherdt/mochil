package net.kdhmedia.mochil.audio

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

class MusicQueueViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = MusicQueueClient()
    private val requests = MusicQueueRequests(
        viewModelScope, credential != null,
        load = { withContext(Dispatchers.IO) { client.get(requireNotNull(credential)) } },
        save = { tracks -> withContext(Dispatchers.IO) { client.put(requireNotNull(credential), tracks) } },
    )
    val state: StateFlow<MusicQueueState> = requests.state

    fun ensureLoaded() {
        if (state.value.loaded || state.value.loading) return
        refresh()
    }

    fun refresh() = requests.refresh()

    fun replace(tracks: List<AudioTrack>) {
        persist(tracks.distinctBy(AudioTrack::identityKey).take(10_000))
    }
    fun append(track: AudioTrack) = requests.edit { queueWithAppended(it, track) }
    fun playNext(track: AudioTrack, current: AudioTrack?) = requests.edit { queueWithNext(it, track, current) }
    fun appendAll(tracks: List<AudioTrack>) {
        val snapshot = tracks.toList()
        requests.edit { queueWithAppendedAll(it, snapshot) }
    }
    fun playNextAll(tracks: List<AudioTrack>, current: AudioTrack?) {
        val snapshot = tracks.toList()
        requests.edit { queueWithNextAll(it, snapshot, current) }
    }
    fun remove(index: Int) = persist(state.value.tracks.filterIndexed { itemIndex, _ -> itemIndex != index })
    fun move(index: Int, direction: Int) {
        val tracks = state.value.tracks.toMutableList()
        if (index !in tracks.indices) return
        val target = (index + direction).coerceIn(0, tracks.lastIndex)
        if (target == index) return
        val item = tracks.removeAt(index)
        tracks.add(target, item)
        persist(tracks)
    }
    fun clear() = persist(emptyList())

    private fun persist(tracks: List<AudioTrack>) = requests.persist(tracks)
}

internal fun queueWithAppended(queue: List<AudioTrack>, track: AudioTrack): List<AudioTrack> =
    (queue.filterNot { it.queueKey() == track.queueKey() } + track).take(MAX_QUEUE_TRACKS)

internal fun queueWithNext(queue: List<AudioTrack>, track: AudioTrack, current: AudioTrack?): List<AudioTrack> {
    val remaining = queue.filterNot { it.queueKey() == track.queueKey() }.toMutableList()
    val currentIndex = current?.let { playing -> remaining.indexOfFirst { it.queueKey() == playing.queueKey() } } ?: -1
    remaining.add((currentIndex + 1).coerceIn(0, remaining.size), track)
    return remaining.take(MAX_QUEUE_TRACKS)
}

internal fun queueWithAppendedAll(queue: List<AudioTrack>, additions: List<AudioTrack>): List<AudioTrack> {
    val keys = additions.mapTo(mutableSetOf()) { it.queueKey() }
    return (queue.filterNot { it.queueKey() in keys } + additions.distinctBy { it.queueKey() }).take(MAX_QUEUE_TRACKS)
}

internal fun queueWithNextAll(queue: List<AudioTrack>, additions: List<AudioTrack>, current: AudioTrack?): List<AudioTrack> {
    val distinct = additions.distinctBy { it.queueKey() }
    val keys = distinct.mapTo(mutableSetOf()) { it.queueKey() }
    val remaining = queue.filterNot { it.queueKey() in keys }.toMutableList()
    val currentIndex = current?.let { playing -> remaining.indexOfFirst { it.queueKey() == playing.queueKey() } } ?: -1
    remaining.addAll((currentIndex + 1).coerceIn(0, remaining.size), distinct)
    return remaining.take(MAX_QUEUE_TRACKS)
}

private fun AudioTrack.queueKey() = identityKey
private const val MAX_QUEUE_TRACKS = 10_000
