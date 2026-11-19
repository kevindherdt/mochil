package net.kdhmedia.mochil.playback

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class ReplayGainCalculatorTest {
    @Test fun offAndMissingMetadataKeepUnityGain() {
        assertEquals(1f, ReplayGainCalculator.multiplier(ReplayGainMode.Off, ReplayGainMetadata()), 0.0001f)
        assertEquals(1f, ReplayGainCalculator.multiplier(ReplayGainMode.Track, ReplayGainMetadata()), 0.0001f)
    }

    @Test fun trackAndAlbumModesChooseTheirOwnMetadata() {
        val metadata = ReplayGainMetadata(trackGainDb = -6.0, albumGainDb = -12.0)
        assertEquals(0.501f, ReplayGainCalculator.multiplier(ReplayGainMode.Track, metadata), 0.002f)
        assertEquals(0.251f, ReplayGainCalculator.multiplier(ReplayGainMode.Album, metadata), 0.002f)
    }

    @Test fun peakPreventionLimitsPositiveGain() {
        val value = ReplayGainCalculator.multiplier(ReplayGainMode.Track, ReplayGainMetadata(trackGainDb = 6.0, trackPeak = 0.8))
        assertEquals(1.25f, value, 0.001f)
        assertTrue(value < 2f)
    }
}
