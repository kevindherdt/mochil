package net.kdhmedia.mochil.audio

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/** One cancellable catalogue context; downloads and account mutations are not children. */
internal class MusicCatalogueRequests(private val scope: CoroutineScope) {
    private val generation = MusicRequestGeneration()
    private var context = SupervisorJob(scope.coroutineContext[Job])

    fun capture(): Long = generation.capture()
    fun isCurrent(candidate: Long): Boolean = generation.isCurrent(candidate)

    fun advance(): Long {
        val next = generation.advance()
        context.cancel()
        context = SupervisorJob(scope.coroutineContext[Job])
        return next
    }

    fun launch(block: suspend CoroutineScope.() -> Unit): Job = scope.launch(context, block = block)
}
