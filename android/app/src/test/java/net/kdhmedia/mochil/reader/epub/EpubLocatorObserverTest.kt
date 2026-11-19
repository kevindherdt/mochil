package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class EpubLocatorObserverTest {
    @Test fun saveFailureFallsBackToDeviceWithoutLosingLastSuccessfulSyncTime() {
        val failure = locatorSaveFailure(ReaderSyncStatus(ReaderSyncPhase.Syncing, 42_000L))

        assertEquals(ReaderSyncPhase.DeviceOnly, failure.status.phase)
        assertEquals(42_000L, failure.status.lastSynchronizedAtMillis)
        assertEquals("This position could not be saved. Reading can continue safely.", failure.message)
        assertTrue(failure.messageIsError)
    }
}
