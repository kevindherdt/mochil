package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class VideoPlaybackQueueTest {
    @Test fun `episode queue keeps bounded neighbors and current index`() {
        val items = (1..80).map(::episode)
        val (window, index) = videoPlaybackWindow(items, items[39], maximumEntries = 20)
        assertEquals(20, window.size)
        assertEquals(items[39].id, window[index].id)
        assertEquals(items[34].id, window.first().id)
    }

    @Test fun `replaying a finished episode starts it instead of advancing to the next`() {
        val fourth = VideoPlaybackEntry("anime", "episode-4", "Episode 4", "url-4", "", 1_421_097, 1421.097, "", finished = true)
        val fifth = fourth.copy(itemId = "episode-5", url = "url-5", positionMs = 13_618, finished = false)
        val queue = VideoPlaybackQueue(listOf(fourth, fifth), 0)
        assertEquals(0L, queue.startPositionMs())
        assertEquals(1_421_097L, queue.entries[0].positionMs) // preserve the server progress snapshot
        assertEquals(13_618L, queue.copy(currentIndex = 1).startPositionMs())
    }

    @Test fun `movies do not unexpectedly autoplay the catalogue`() {
        val movie = episode(1).copy(show = "")
        val (window, index) = videoPlaybackWindow(listOf(movie, episode(2).copy(show = "")), movie)
        assertEquals(listOf(movie), window)
        assertEquals(0, index)
    }

    @Test fun `queue codec rejects oversized payloads`() {
        val entry = VideoPlaybackEntry("library", "item", "Episode", "https://example.test/video", "", 12, 30.0, "", "revision-2")
        val encoded = encodeVideoPlaybackQueue(VideoPlaybackQueue(listOf(entry), 0))
        assertEquals(entry, decodeVideoPlaybackQueue(encoded)?.entries?.single())
        assertNull(decodeVideoPlaybackQueue("x".repeat(1025), maximumBytes = 1024))
    }

    @Test fun `preview positions are bounded and quantized`() {
        assertEquals(0, quantizedVideoPreviewSeconds(0, 100.0))
        assertEquals(20, quantizedVideoPreviewSeconds(27_900, 100.0))
        assertEquals(90, quantizedVideoPreviewSeconds(120_000, 100.0))
        assertNull(quantizedVideoPreviewSeconds(2_000, 0.0))
    }

    @Test fun `preview path retains identity and revision`() {
        val entry = VideoPlaybackEntry("TV Series", "folder/episode 1", "Episode", "url", "", 0, 100.0, "", "rev 2")
        assertEquals("/api/android/v1/video-previews/TV%20Series/folder%2Fepisode%201/20.jpg?v=rev%202", videoPreviewPath(entry, 20))
    }

    private fun episode(number: Int) = VideoItem("id-$number", "library", "Episode $number", "Show", 1, number, 1200.0, "", "/video/$number", "", 0.0, emptyList())
}
