package net.kdhmedia.mochil.dashboard

import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.video.VideoItem

data class ContinueReading(val book: BookSummary, val fraction: Double, val updatedAt: Long)
data class ContinueListening(val track: AudioTrack, val positionSeconds: Double, val durationSeconds: Double)
data class DashboardSnapshot(
    val reading: List<ContinueReading>,
    val listening: List<ContinueListening>,
    val watching: List<VideoItem>,
)

data class DashboardState(
    val reading: List<ContinueReading> = emptyList(),
    /** The newest books across the first browsable library, for Discover's "Recently added" rail (`?sort=recent`). */
    val recent: List<BookSummary> = emptyList(),
    val listening: List<ContinueListening> = emptyList(),
    val watching: List<VideoItem> = emptyList(),
    /** Books the reader put on hold or dropped: out of Continue reading, kept here with a way back. */
    val held: List<ContinueReading> = emptyList(),
    /** Books the reader finished, newest first, each with a way back into reading. */
    val finished: List<ContinueReading> = emptyList(),
    val loading: Boolean = true,
    val error: String? = null,
)
