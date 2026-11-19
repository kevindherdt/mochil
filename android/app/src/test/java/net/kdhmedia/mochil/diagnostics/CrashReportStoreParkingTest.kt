package net.kdhmedia.mochil.diagnostics

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

/**
 * Where a crash used to be a permanent problem: the app parked on the sign-in screen on *every*
 * start while a report was pending, so a single crash meant re-entering the server address each time
 * it was opened. It must park exactly once per crash, and never after the person has seen the report.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [34])
class CrashReportStoreParkingTest {

    private fun store() = CrashReportStore(ApplicationProvider.getApplicationContext<Context>()).also { it.clear() }

    private fun crash(): RuntimeException = RuntimeException("boom").apply {
        stackTrace = arrayOf(StackTraceElement("a.B", "c", "B.kt", 1))
    }

    @Test
    fun noCrashMeansNoParking() {
        val store = store()
        assertFalse(store.aFreshCrashShouldStartSignedOut())
    }

    @Test
    fun aCrashParksOnceAndThenLetsTheAppResume() {
        val store = store()
        store.capture(crash())
        assertNotNull(store.pending())
        assertTrue("the first start after a crash must not resume", store.aFreshCrashShouldStartSignedOut())
        assertFalse("the second start must resume", store.aFreshCrashShouldStartSignedOut())
        assertFalse(store.aFreshCrashShouldStartSignedOut())
    }

    @Test
    fun aSecondCrashParksAgain() {
        val store = store()
        store.capture(crash())
        assertTrue(store.aFreshCrashShouldStartSignedOut())
        store.capture(crash())
        assertTrue("a new crash is a new reason to stay signed out", store.aFreshCrashShouldStartSignedOut())
    }

    @Test
    fun dismissingTheReportDoesNotParkAgain() {
        val store = store()
        store.capture(crash())
        assertTrue(store.aFreshCrashShouldStartSignedOut())
        store.clear()
        assertFalse(store.hasPending())
        assertFalse(store.aFreshCrashShouldStartSignedOut())
    }
}
