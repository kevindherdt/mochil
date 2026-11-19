package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class VideoClientRedirectTest {
    @Test fun `show catalogue follows safe Android API redirect`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-shows") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-video-shows")
            exchange.sendResponseHeaders(303, -1); exchange.close()
        }
        server.createContext("/api/android/v1/canonical-video-shows") { exchange ->
            val body = """[{"name":"Show","source_name":"Show","item_id":"episode","count":2}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            assertEquals(listOf("Show"), VideoClient().shows(credential, "tv").items.map(VideoShow::name))
        } finally { server.stop(0) }
    }
}
