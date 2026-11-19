package net.kdhmedia.mochil.network

import net.kdhmedia.mochil.auth.DeviceSessionEvents
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AndroidApiAuthorizationTest {
    @Test fun `401 is recognized as an expired device session and nothing else is`() {
        try {
            for (status in listOf(200, 204, 301, 400, 403, 404, 409, 500, 503)) {
                assertFalse("status $status must not be treated as expired", reportIfExpiredDeviceSession(status))
            }
            assertTrue(reportIfExpiredDeviceSession(401))
        } finally { DeviceSessionEvents.connected() }
    }

    @Test fun `a 401 anywhere notifies DeviceSessionEvents exactly once until reconnected`() {
        DeviceSessionEvents.connected()
        try {
            val before = DeviceSessionEvents.generation.value
            reportIfExpiredDeviceSession(500)
            assertEquals("a non-401 must not bump the generation", before, DeviceSessionEvents.generation.value)
            reportIfExpiredDeviceSession(401)
            assertEquals("a 401 must bump the generation exactly once", before + 1, DeviceSessionEvents.generation.value)
            reportIfExpiredDeviceSession(401)
            assertEquals("a second 401 before reconnecting must not bump it again", before + 1, DeviceSessionEvents.generation.value)
        } finally { DeviceSessionEvents.connected() }
    }
}
