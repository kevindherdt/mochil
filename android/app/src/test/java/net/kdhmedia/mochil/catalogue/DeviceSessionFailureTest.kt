package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DeviceSessionFailureTest {
    @Test fun `unauthorized catalogue response expires device session`() = assertTrue(CatalogueException(401, "Unauthorized").isExpiredDeviceSession())
    @Test fun `server and network failures do not revoke local session`() {
        assertFalse(CatalogueException(500, "Unavailable").isExpiredDeviceSession())
        assertFalse(java.net.SocketTimeoutException().isExpiredDeviceSession())
    }
}
