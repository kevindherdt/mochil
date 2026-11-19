package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Test

class ReaderSyncStatusTest {
    private val clockLabel: (Long) -> String = { "14:32" }

    @Test fun `synced status includes the acknowledgement time`() {
        assertEquals("Synced · 14:32", readerSyncLabel(ReaderSyncStatus(ReaderSyncPhase.Synced, 1), clockLabel))
    }

    @Test fun `pending status makes device-only safety explicit`() {
        assertEquals("Saved on device · Last sync 14:32", readerSyncLabel(ReaderSyncStatus(ReaderSyncPhase.DeviceOnly, 1), clockLabel))
    }

    @Test fun `syncing status remains compact`() {
        assertEquals("Syncing…", readerSyncLabel(ReaderSyncStatus(ReaderSyncPhase.Syncing, 1), clockLabel))
    }
}
