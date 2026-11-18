package net.kdhmedia.mochil.video

import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class VideoEmptyArtworkTest {
    @Test fun blankArtworkDoesNotResolveToServerRoot() {
        val credential = DeviceCredential("https://example.test", "device", "token")
        assertEquals("", VideoClient().absolute(credential, ""))
        assertEquals("", VideoClient().absolute(credential, "  "))
    }
}
