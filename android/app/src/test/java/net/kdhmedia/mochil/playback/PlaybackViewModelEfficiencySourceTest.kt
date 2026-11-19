package net.kdhmedia.mochil.playback

import java.nio.file.Files
import java.nio.file.Paths
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlaybackViewModelEfficiencySourceTest {
    @Test fun `media preparation precedes mutations of existing queue`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/playback/PlaybackViewModel.kt"))
        for (name in listOf("fun playNext(track:", "fun addToQueue(track:", "private fun insertTracks(")) {
            val body = source.substringAfter(name).substringBefore("\n    }")
            val preparation = body.indexOf("toPlaybackMediaItem(streamUrl(")
            val removal = body.indexOf("removeQueuedCop")
            assertTrue(name, preparation >= 0 && removal > preparation)
            assertTrue(name, body.indexOf("queueByKey =") > preparation)
        }
        val play = source.substringAfter("fun play(tracks:").substringBefore("\n    }")
        assertTrue(play.indexOf("val items =") < play.indexOf("queueByKey ="))
    }

    @Test fun `playback projections and clock stop when they have no active consumer`() {
        val source = Files.readString(Paths.get("src/main/java/net/kdhmedia/mochil/playback/PlaybackViewModel.kt"))
        assertTrue(source.contains("SharingStarted.WhileSubscribed"))
        assertFalse(source.contains("SharingStarted.Eagerly"))
        assertTrue(source.contains("positionTicker = if (player.isPlaying)"))
    }
}
