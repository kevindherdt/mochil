package net.kdhmedia.mochil.video

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/** Main-thread owner: invalidate old results now, defer only the next network request. */
internal class VideoSearchDebounce(
    private val scope: CoroutineScope,
    private val requests: VideoRequestGeneration,
    private val pause: suspend () -> Unit = { delay(250) },
) {
    private var job: Job? = null

    fun submit(load: () -> Unit) {
        job?.cancel()
        val generation = requests.advance()
        job = scope.launch {
            pause()
            // Navigation may have loaded another library or season during the delay.
            if (requests.isCurrent(generation)) load()
        }
    }
}
