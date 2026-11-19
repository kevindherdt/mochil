package net.kdhmedia.mochil.video

import org.junit.Assert.assertEquals
import org.junit.Test

/** Characterizes VideoPlayerActivity.pictureInPictureParams' aspect-ratio math before extraction. */
class VideoPictureInPictureAspectRatioTest {
    @Test fun usesTheVideoWidthToHeightRatioWhenWithinBounds() {
        assertEquals(16.0 / 9.0, videoPictureInPictureAspectRatio(1920, 1080), 1e-9)
    }

    @Test fun fallsBackToSixteenByNineWhenDimensionsAreMissingOrZero() {
        assertEquals(16.0 / 9.0, videoPictureInPictureAspectRatio(0, 0), 1e-9)
    }

    @Test fun onlyTheMissingDimensionFallsBackLeavingTheOtherAsGiven() {
        // Characterizes the original takeIf-per-dimension behavior: a non-positive width defaults
        // to 16 independently of a still-valid height (and vice versa), not a joint 16:9 fallback.
        assertEquals(videoPictureInPictureAspectRatio(16, 1080), videoPictureInPictureAspectRatio(-5, 1080), 1e-9)
        assertEquals(videoPictureInPictureAspectRatio(1920, 9), videoPictureInPictureAspectRatio(1920, 0), 1e-9)
    }

    @Test fun clampsExtremelyWideVideoToTheMaximumSupportedRatio() {
        assertEquals(2.39, videoPictureInPictureAspectRatio(100_000, 100), 1e-9)
    }

    @Test fun clampsExtremelyTallVideoToTheMinimumSupportedRatio() {
        assertEquals(1.0 / 2.39, videoPictureInPictureAspectRatio(100, 100_000), 1e-9)
    }
}
