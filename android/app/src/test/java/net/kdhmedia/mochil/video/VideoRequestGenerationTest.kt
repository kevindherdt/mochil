package net.kdhmedia.mochil.video

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.assertEquals
import kotlinx.coroutines.CancellationException
import org.junit.Test

class VideoRequestGenerationTest {
    @Test fun onlyCurrentNonCancellationFailuresReachTheScreen() {
        val requests = VideoRequestGeneration()
        requests.advance()
        val mutation = requests.snapshot()
        var errors = 0
        requests.onCurrentFailure(mutation, IllegalStateException()) { errors++ }
        assertEquals(1, errors)
        requests.onCurrentFailure(mutation, CancellationException()) { errors++ }
        assertEquals(1, errors)
        requests.advance()
        requests.onCurrentFailure(mutation, IllegalStateException()) { errors++ }
        assertEquals(1, errors)
    }

    @Test fun navigationInvalidatesAnOlderVideoRequest() {
        val requests = VideoRequestGeneration()
        val library = requests.advance()
        val season = requests.advance()
        assertFalse(requests.isCurrent(library))
        assertTrue(requests.isCurrent(season))
    }
}
