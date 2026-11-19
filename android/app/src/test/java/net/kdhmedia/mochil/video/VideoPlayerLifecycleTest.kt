package net.kdhmedia.mochil.video

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoPlayerLifecycleTest {
    @Test fun playingVideoMayEnterPictureInPictureForOrdinaryAppNavigation() {
        assertTrue(shouldEnterVideoPictureInPicture(closeRequested = false, isPlaying = true))
    }

    @Test fun explicitCloseNeverEntersPictureInPicture() {
        assertFalse(shouldEnterVideoPictureInPicture(closeRequested = true, isPlaying = true))
    }

    @Test fun pausedVideoDoesNotEnterPictureInPicture() {
        assertFalse(shouldEnterVideoPictureInPicture(closeRequested = false, isPlaying = false))
    }
}
