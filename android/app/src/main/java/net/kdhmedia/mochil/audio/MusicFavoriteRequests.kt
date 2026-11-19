package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/** Main-thread owner of optimistic favorites, ordered writes and snapshot admission. */
internal class MusicFavoriteRequests(
    private val scope: CoroutineScope,
    private val read: suspend () -> List<AudioTrack>,
    private val write: suspend (AudioTrack, Boolean) -> Unit,
    private val current: () -> MusicState,
    private val publish: (MusicState) -> Unit,
    private val notify: (String) -> Unit,
) {
    private val writes = Mutex()
    private val reads = MusicRequestGeneration()
    private var nextMutation = 0L
    private val pending = mutableMapOf<String, Long>()
    private val confirmed = mutableMapOf<String, Boolean>()

    fun refresh() {
        if (pending.isNotEmpty()) return
        val generation = reads.advance()
        scope.launch {
            try {
                val tracks = read()
                if (reads.isCurrent(generation) && pending.isEmpty()) {
                    publish(current().copy(favoriteTracks = tracks,
                        favoriteIds = tracks.mapTo(mutableSetOf(), AudioTrack::identityKey)))
                }
            } catch (cancelled: CancellationException) { throw cancelled }
            catch (_: Exception) { /* A failed refresh must not erase usable favorites. */ }
        }
    }

    fun toggle(track: AudioTrack) {
        val key = track.identityKey
        val before = current()
        val desired = key !in before.favoriteIds
        confirmed.putIfAbsent(key, key in before.favoriteIds)
        val mutation = ++nextMutation
        pending[key] = mutation
        reads.advance()
        publish(before.withFavorite(track, desired))
        scope.launch {
            writes.withLock {
                var failed = false
                try {
                    write(track, desired)
                    confirmed[key] = desired
                } catch (cancelled: CancellationException) { throw cancelled }
                catch (_: Exception) { failed = true }
                if (pending[key] == mutation) {
                    if (failed) {
                        publish(current().withFavorite(track, confirmed[key] == true))
                        notify("Could not update this favorite.")
                    } else {
                        publish(current().withFavorite(track, desired))
                        notify(if (desired) "Added to favorites." else "Removed from favorites.")
                    }
                    pending.remove(key)
                    confirmed.remove(key)
                }
            }
            if (pending.isEmpty()) refresh()
        }
    }
}
