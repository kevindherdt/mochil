package net.kdhmedia.mochil.playback

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ProgressOwnershipPolicyTest {
    @Test fun `passive stale writer cannot rewind or unfinish remote progress`() {
        assertFalse(shouldUploadPassiveProgress(600.0, false, 120.0, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(1_200.0, true, 1_199.0, 1_200.0, localFinished = true))
    }

    @Test fun `passive writer only writes a meaningful advance`() {
        assertTrue(shouldUploadPassiveProgress(120.0, false, 300.0, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(300.0, false, 300.0, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(300.2, false, 300.0, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(300.0, false, 300.2, 1_200.0))
        assertTrue(shouldUploadPassiveProgress(300.0, false, 300.3, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(300.3, false, 300.0, 1_200.0))
        assertFalse(shouldUploadPassiveProgress(304.0, false, 300.0, 1_200.0))
    }

    @Test fun `completion is saved even when its final clock sample is equivalent`() {
        assertTrue(shouldUploadPassiveProgress(98.0, false, 98.1, 100.0, localFinished = true))
    }

    @Test fun `periodic persistence requires actively advancing media`() {
        assertTrue(shouldPersistPeriodicProgress(isPlaying = true, hasCurrentItem = true))
        assertFalse(shouldPersistPeriodicProgress(isPlaying = false, hasCurrentItem = true))
        assertFalse(shouldPersistPeriodicProgress(isPlaying = true, hasCurrentItem = false))
    }

    @Test fun `short audio is not finished merely because less than thirty seconds remain`() {
        assertFalse(audioProgressFinished(0.0, 20.0))
        assertFalse(audioProgressFinished(19.0, 20.0))
        assertTrue(audioProgressFinished(19.6, 20.0))
    }

    @Test fun `ordinary audio follows the server ninety eight percent boundary`() {
        assertFalse(audioProgressFinished(175.0, 180.0))
        assertTrue(audioProgressFinished(176.4, 180.0))
    }
}
