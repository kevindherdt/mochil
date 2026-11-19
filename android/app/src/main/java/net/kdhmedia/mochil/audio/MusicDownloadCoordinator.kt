package net.kdhmedia.mochil.audio

import net.kdhmedia.mochil.auth.DeviceCredential
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/** Owns offline-track selection and file mutations outside UI state orchestration. */
internal class MusicDownloadCoordinator(
    private val store: AudioDownloadStore,
    private val streamUrl: (DeviceCredential, AudioTrack) -> String,
) {
    fun ids(): Set<String> = store.ids()
    fun tracks(libraryId: String? = null): List<AudioTrack> = store.tracks(libraryId)
    fun snapshot(libraryId: String? = null): AudioDownloadSnapshot = store.snapshot(libraryId)
    fun playbackUrl(track: AudioTrack): String? = store.playbackUrl(track)

    suspend fun pending(tracks: List<AudioTrack>): List<AudioTrack> = select(tracks, downloaded = false)

    suspend fun downloaded(tracks: List<AudioTrack>): List<AudioTrack> = select(tracks, downloaded = true)

    private suspend fun select(tracks: List<AudioTrack>, downloaded: Boolean): List<AudioTrack> = withContext(Dispatchers.IO) {
        if (tracks.isEmpty()) return@withContext emptyList()
        val snapshot = store.snapshot()
        val podcastKeys = snapshot.tracks.filter { it.libraryKind == "podcasts" && it.feedId.isNotBlank() }
            .mapTo(hashSetOf()) { it.feedId to it.id }
        unique(tracks).filter { track ->
            val exists = track.downloadKey in snapshot.ids ||
                (track.libraryKind == "podcasts" && track.feedId.isNotBlank() && (track.feedId to track.id) in podcastKeys)
            exists == downloaded
        }
    }

    fun toggle(credential: DeviceCredential?, track: AudioTrack): Boolean {
        return if (store.contains(track)) {
            if (!store.remove(track)) error("Could not remove this download.")
            false
        } else {
            val saved = credential ?: error("This device is no longer connected.")
            store.download(saved, track, streamUrl(saved, track))
            true
        }
    }

    suspend fun downloadBatch(credential: DeviceCredential, tracks: List<AudioTrack>, onProgress: suspend (Int, AudioTrack) -> Unit): MusicDownloadBatchResult {
        var completed = 0
        for (track in tracks) {
            currentCoroutineContext().ensureActive()
            val failure = runCatching { store.download(credential, track, streamUrl(credential, track)) }.exceptionOrNull()
            if (failure != null) return MusicDownloadBatchResult(completed, failure)
            completed++
            onProgress(completed, track)
        }
        return MusicDownloadBatchResult(completed)
    }

    suspend fun removeBatch(tracks: List<AudioTrack>, onProgress: suspend (Int, AudioTrack) -> Unit): MusicDownloadBatchResult {
        var completed = 0
        for (track in tracks) {
            currentCoroutineContext().ensureActive()
            val failure = runCatching {
                if (!store.remove(track)) error("Could not remove ${track.title}.")
            }.exceptionOrNull()
            if (failure != null) return MusicDownloadBatchResult(completed, failure)
            completed++
            onProgress(completed, track)
        }
        return MusicDownloadBatchResult(completed)
    }

    private fun unique(tracks: List<AudioTrack>) = tracks.distinctBy(AudioTrack::downloadKey)
}

internal data class MusicDownloadBatchResult(val completed: Int, val failure: Throwable? = null)

interface AudioDownloadStore {
    fun ids(): Set<String>
    fun tracks(libraryId: String? = null): List<AudioTrack>
    fun snapshot(libraryId: String? = null): AudioDownloadSnapshot = AudioDownloadSnapshot(ids(), tracks(libraryId))
    fun contains(track: AudioTrack): Boolean = track.downloadKey in ids()
    fun playbackUrl(track: AudioTrack): String?
    fun download(credential: DeviceCredential, track: AudioTrack, url: String)
    fun remove(track: AudioTrack): Boolean
}

data class AudioDownloadSnapshot(val ids: Set<String>, val tracks: List<AudioTrack>)

internal val AudioTrack.downloadKey: String get() = identityKey

internal fun sameOfflineAudioIdentity(stored: AudioTrack, requested: AudioTrack): Boolean =
    stored.downloadKey == requested.downloadKey || (
        stored.libraryKind == "podcasts" &&
            requested.libraryKind == "podcasts" &&
            stored.feedId.isNotBlank() &&
            stored.feedId == requested.feedId &&
            stored.id == requested.id
        )
