package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Test

class PlaybackClockTest {
    @Test fun shortMediaUsesMinutesAndSeconds() {
        assertEquals("0:00", playbackClock(-1))
        assertEquals("3:05", playbackClock(185_999))
    }

    @Test fun longMediaUsesHoursMinutesAndSeconds() {
        assertEquals("1:00:00", playbackClock(3_600_000))
        assertEquals("12:34:56", playbackClock(45_296_000))
    }

    @Test fun shuffleIsOnlyShownForOrdinaryMusic() {
        assertEquals(true, showsShuffle("music"))
        assertEquals(false, showsShuffle("podcasts"))
        assertEquals(false, showsShuffle("audiobooks"))
    }

    @Test fun playbackSpeedIsOnlyShownForSpokenMedia() {
        assertEquals(false, showsPlaybackSpeed("music"))
        assertEquals(true, showsPlaybackSpeed("podcasts"))
        assertEquals(true, showsPlaybackSpeed("audiobooks"))
        assertEquals("1", playbackSpeedLabel(1f))
        assertEquals("1.25", playbackSpeedLabel(1.25f))
    }

    @Test fun sourceFactsDescribeTheEffectiveStream() {
        val original = AudioTrack("id", "music", "Track", "Artist", "Album", 180.0, container = "flac", sampleRate = 44_100, bitDepth = 24)
        assertEquals("FLAC · 24-bit · 44.1 kHz", playerSourceFacts(original))
        assertEquals("AAC · 128 kbps · Converted", playerSourceFacts(original.copy(playbackMode = "converted", playbackQuality = "medium")))
    }
}
