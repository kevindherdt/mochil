package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.audio.AudioTrack
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class PlaybackPresentationStateTest {
    private val track = AudioTrack("track", "music", "Title", "Artist", "Album", 180.0)

    @Test fun timelineTicksDoNotInvalidateTvDockChrome() {
        val before = PlaybackState(current = track, isPlaying = true, positionMs = 1_000, durationMs = 180_000)
        val after = before.copy(positionMs = 2_000)

        assertEquals(before.toDockState(), after.toDockState())
        assertEquals(before.toTransportState(), after.toTransportState())
        assertEquals(before.toPlayerState(), after.toPlayerState())
        assertNotEquals(before.toTimelineState(), after.toTimelineState())
    }

    @Test fun trackAndTransportChangesStillInvalidateTvDockChrome() {
        val before = PlaybackState(current = track, isPlaying = true)

        assertNotEquals(before.toDockState(), before.copy(isPlaying = false).toDockState())
        assertNotEquals(before.toDockState(), before.copy(current = track.copy(id = "next")).toDockState())
        assertNotEquals(before.toTransportState(), before.copy(repeatMode = 1).toTransportState())
        assertNotEquals(before.toTransportState(), before.copy(shuffleEnabled = true).toTransportState())
        assertNotEquals(before.toTransportState(), before.copy(speed = 1.25f).toTransportState())
        assertNotEquals(before.toPlayerState(), before.copy(sleepRemainingMs = 60_000).toPlayerState())
    }
}
