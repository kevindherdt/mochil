package net.kdhmedia.mochil.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Mochil is normally reached on the LAN or over Tailscale, so a plain http:// private address has to
 * work. A public host over http does not: the app says why instead of silently failing at connect.
 */
class ServerAddressTest {
    @Test
    fun `private addresses are accepted over http`() {
        listOf(
            "http://192.168.50.14:8099",
            "http://10.0.0.5",
            "http://172.20.1.4:8080",
            "http://169.254.10.2",
            "http://100.106.121.38:8099", // Tailscale
            "http://nixos:8099",
            "http://mochil.local",
            "http://mochil.ts.net",
            "http://[fd7a:115c:a1e0::1]:8099",
        ).forEach { assertEquals(it, ServerAddress.normalize(it)) }
    }

    @Test
    fun `https is accepted anywhere`() {
        assertEquals("https://media.example.com", ServerAddress.normalize("https://media.example.com"))
        assertEquals("https://books.example.com", ServerAddress.normalize("  https://books.example.com/ "))
    }

    @Test
    fun `a public host over http is refused with a reason`() {
        val error = runCatching { ServerAddress.normalize("http://media.example.com") }.exceptionOrNull()
        assertTrue(error is LoginException)
        assertTrue((error as LoginException).message!!.contains("https://"))
    }

    @Test
    fun `addresses without a scheme are refused`() {
        assertTrue(runCatching { ServerAddress.normalize("192.168.1.10:8099") }.exceptionOrNull() is LoginException)
        assertTrue(runCatching { ServerAddress.normalize("ftp://192.168.1.10") }.exceptionOrNull() is LoginException)
    }

    @Test
    fun `hosts with credentials or queries are refused`() {
        assertTrue(runCatching { ServerAddress.normalize("http://user:pw@192.168.1.10") }.exceptionOrNull() is LoginException)
        assertTrue(runCatching { ServerAddress.normalize("http://192.168.1.10/?next=x") }.exceptionOrNull() is LoginException)
    }

    @Test
    fun `classifying hosts`() {
        assertTrue(ServerAddress.isPrivateHost("192.168.1.1"))
        assertFalse(ServerAddress.isPrivateHost("8.8.8.8"))
        assertFalse(ServerAddress.isPrivateHost("100.200.1.1")) // outside the CGNAT block
        assertFalse(ServerAddress.isPrivateHost("media.example.com"))
    }
}
