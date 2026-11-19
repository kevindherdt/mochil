package net.kdhmedia.mochil.podcast

/** Ensures only the newest explicit podcast-play request may start playback. */
internal class PodcastPlaybackResolution {
    private var generation = 0L

    fun begin(): Long = ++generation

    fun cancel() {
        generation++
    }

    fun isCurrent(ticket: Long): Boolean = ticket == generation
}
