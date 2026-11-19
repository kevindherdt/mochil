package net.kdhmedia.mochil.audio

/** Invalidates asynchronous content work whenever the visible music context changes. */
internal class MusicRequestGeneration {
    private var generation = 0L

    fun advance(): Long {
        generation = if (generation == Long.MAX_VALUE) 1L else generation + 1L
        return generation
    }

    fun capture(): Long = generation

    fun isCurrent(candidate: Long): Boolean = candidate == generation
}
