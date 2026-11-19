package net.kdhmedia.mochil.video

import kotlinx.coroutines.CancellationException

/** Invalidates video catalogue work when library, query, show, or season changes. */
internal class VideoRequestGeneration {
    private var generation = 0L

    fun advance(): Long {
        generation = if (generation == Long.MAX_VALUE) 1L else generation + 1L
        return generation
    }

    fun isCurrent(candidate: Long): Boolean = candidate == generation

    fun snapshot(): Long = generation

    fun onCurrentFailure(candidate: Long, failure: Throwable, report: () -> Unit) {
        if (failure !is CancellationException && isCurrent(candidate)) report()
    }
}
