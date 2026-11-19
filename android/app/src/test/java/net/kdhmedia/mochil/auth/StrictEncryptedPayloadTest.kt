package net.kdhmedia.mochil.auth

import org.junit.Assert.*
import org.junit.Test

class StrictEncryptedPayloadTest {
    @Test fun `absent envelope is empty without attempting decryption`() {
        assertNull(requireReadableEncryptedPayload(null, null) { error("must not read") })
    }

    @Test fun `incomplete or unreadable envelope is not treated as absence`() {
        for ((payload, iv) in listOf("payload" to null, null to "iv", "payload" to "iv")) {
            assertThrows(IllegalStateException::class.java) {
                requireReadableEncryptedPayload(payload, iv) { null }
            }
        }
        val bytes = byteArrayOf(1, 2)
        assertSame(bytes, requireReadableEncryptedPayload("payload", "iv") { bytes })
    }
}
