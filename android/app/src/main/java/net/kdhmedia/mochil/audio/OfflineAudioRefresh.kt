package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/** Owns refresh ordering on the caller's UI scope; storage runs in the supplied loader. */
internal class OfflineAudioRefresh(
    private val scope: CoroutineScope,
    private val load: suspend (String?) -> AudioDownloadSnapshot,
    private val publish: (String?, AudioDownloadSnapshot) -> Unit,
    private val onFailure: (String?, Exception) -> Unit,
) {
    private val requests = MusicRequestGeneration()
    private var job: Job? = null

    fun refresh(libraryId: String?) {
        val generation = requests.advance()
        job?.cancel()
        job = scope.launch {
            val snapshot = try {
                load(libraryId)
            } catch (cancelled: CancellationException) {
                throw cancelled
            } catch (failure: Exception) {
                if (requests.isCurrent(generation)) onFailure(libraryId, failure)
                return@launch
            }
            if (requests.isCurrent(generation)) publish(libraryId, snapshot)
        }
    }
}
