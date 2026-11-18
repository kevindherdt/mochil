package net.kdhmedia.mochil.hermes

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CaptureRunGateTest {
    @Test
    fun stoppingAndRestartingInvalidatesThePreviousCaptureWorker() {
        val gate = CaptureRunGate()
        val previousRun = gate.start()

        gate.stop()
        val currentRun = gate.start()

        assertFalse(gate.isCurrent(previousRun))
        assertTrue(gate.isCurrent(currentRun))
    }

    @Test
    fun captureStartFailureRunsCleanupAndPreservesTheOriginalError() {
        val expected = IllegalStateException("start failed")
        var cleaned = 0

        val actual = runCatching {
            startCaptureOrCleanup(start = { throw expected }, cleanup = { cleaned += 1 })
        }.exceptionOrNull()

        assertTrue(actual === expected)
        assertTrue(cleaned == 1)
    }

    @Test
    fun successfulCaptureStartDoesNotRunFailureCleanup() {
        var cleaned = 0

        val result = startCaptureOrCleanup(start = { "started" }, cleanup = { cleaned += 1 })

        assertTrue(result == "started")
        assertTrue(cleaned == 0)
    }

    @Test
    fun stoppingInvalidatesTheCurrentCaptureWorker() {
        val gate = CaptureRunGate()
        val run = gate.start()

        gate.stop()

        assertFalse(gate.isCurrent(run))
    }
}
