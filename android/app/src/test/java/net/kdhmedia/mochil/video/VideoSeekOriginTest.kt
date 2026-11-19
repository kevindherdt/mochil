package net.kdhmedia.mochil.video

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class VideoSeekOriginTest {
    @Test
    fun matchingRecoverySeekIsNotExplicit() {
        val origin = VideoSeekOrigin()
        origin.expectAutomatic(2, 45_000)

        assertFalse(origin.isExplicit(2, 45_250))
    }

    @Test
    fun automaticExpectationIsConsumedOnce() {
        val origin = VideoSeekOrigin()
        origin.expectAutomatic(0, 45_000)

        assertFalse(origin.isExplicit(0, 45_000))
        assertTrue(origin.isExplicit(0, 45_000))
    }

    @Test
    fun differentSeekRemainsExplicit() {
        val origin = VideoSeekOrigin()
        origin.expectAutomatic(0, 45_000)

        assertTrue(origin.isExplicit(0, 90_000))
    }

    @Test
    fun clearedExpectationCannotHideAViewerSeek() {
        val origin = VideoSeekOrigin()
        origin.expectAutomatic(0, 45_000)
        origin.clear()

        assertTrue(origin.isExplicit(0, 45_000))
    }
}
