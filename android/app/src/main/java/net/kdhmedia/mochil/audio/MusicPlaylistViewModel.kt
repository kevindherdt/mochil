package net.kdhmedia.mochil.audio

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.withContext
import net.kdhmedia.mochil.auth.DeviceCredentialStore

class MusicPlaylistViewModel(application: Application) : AndroidViewModel(application) {
    private val credential = DeviceCredentialStore(application).load()
    private val client = MusicPlaylistClient()
    private val requests = MusicPlaylistRequests(viewModelScope, { name ->
        withContext(Dispatchers.IO) { client.get(requireNotNull(credential), name) }
    }) {
        withContext(Dispatchers.IO) { client.overview(requireNotNull(credential)) }
    }
    val state: StateFlow<MusicPlaylistState> = requests.state

    init { refresh() }

    fun refresh() {
        if (credential == null) return requests.error("This device is no longer connected.")
        requests.refresh()
    }

    fun saveQueue(name: String, tracks: List<AudioTrack>) {
        val saved = credential ?: return
        val cleanName = normalizePlaylistName(name)
        if (cleanName.isBlank() || cleanName.length > 80 || tracks.isEmpty()) {
            requests.error(if (tracks.isEmpty()) "Add tracks to the queue first." else "Use a playlist name between 1 and 80 characters.")
            return
        }
        val snapshot = playlistTracks(tracks)
        mutate("Playlist saved.") {
            val current = try { client.get(saved, cleanName) } catch (_: MusicPlaylistMissing) {
                client.create(saved, cleanName)
                client.get(saved, cleanName)
            }
            client.put(saved, current.name, snapshot, current.revision)
        }
    }

    fun addTrack(name: String, track: AudioTrack) {
        val saved = credential ?: return
        val cleanName = normalizePlaylistName(name)
        if (cleanName.isBlank() || cleanName.length > 80) {
            requests.error("Use a playlist name between 1 and 80 characters.")
            return
        }
        mutate("Added to $cleanName.") { client.add(saved, cleanName, track) }
    }

    fun delete(name: String) {
        val saved = credential ?: return
        val known = state.value.selectedPlaylist?.takeIf { it.name.equals(name, ignoreCase = true) }
        mutate("Playlist deleted.") {
            val current = known ?: client.get(saved, name)
            client.delete(saved, current.name, current.revision)
        }
    }

    fun select(name: String?) = requests.select(name)

    fun removeTrack(name: String, index: Int) {
        val key = selectedTrackKey(name, index) ?: return
        updateTracks(name) { tracks -> tracks.filterNot { it.identityKey == key } }
    }

    fun moveTrack(name: String, index: Int, direction: Int) {
        val key = selectedTrackKey(name, index) ?: return
        updateTracks(name) { tracks ->
            val current = tracks.indexOfFirst { it.identityKey == key }
            if (current < 0) return@updateTracks tracks
            val target = (current + direction).coerceIn(0, tracks.lastIndex)
            if (target == current) return@updateTracks tracks
            tracks.toMutableList().apply { add(target, removeAt(current)) }
        }
    }

    private fun selectedTrackKey(name: String, index: Int) = state.value.selectedPlaylist
        ?.takeIf { it.name.equals(name, ignoreCase = true) }?.tracks?.getOrNull(index)?.identityKey

    suspend fun playbackTracks(name: String): List<AudioTrack> = withContext(Dispatchers.IO) {
        client.get(requireNotNull(credential), name).tracks
    }

    fun artworkUrl(path: String): String = if (path.startsWith("/") && !path.startsWith("//")) credential?.serverUrl.orEmpty() + path else ""

    fun rename(name: String, newName: String) {
        val saved = credential ?: return
        val clean = normalizePlaylistName(newName)
        if (clean.isBlank() || clean.length > 80) return requests.error("Use a playlist name between 1 and 80 characters.")
        val known = state.value.selectedPlaylist?.takeIf { it.name.equals(name, ignoreCase = true) }
        requests.mutate("Playlist renamed.", name to clean) {
            withContext(Dispatchers.IO) {
                val current = known ?: client.get(saved, name)
                client.rename(saved, current.name, clean, current.revision)
            }
        }
    }

    private fun updateTracks(name: String, transform: (List<AudioTrack>) -> List<AudioTrack>) {
        val saved = credential ?: return
        mutate("Playlist updated.") {
            val playlist = client.get(saved, name)
            client.editVisible(saved, playlist, playlistTracks(transform(playlist.tracks)))
        }
    }

    private fun mutate(successMessage: String = "Playlist updated.", action: suspend () -> Unit) {
        requests.mutate(successMessage) {
            withContext(Dispatchers.IO) { action() }
        }
    }
}

internal fun normalizePlaylistName(value: String) = value.trim().replace(Regex("\\s+"), " ")

internal fun playlistTracks(tracks: List<AudioTrack>) = tracks.distinctBy(AudioTrack::identityKey).take(10_000)
