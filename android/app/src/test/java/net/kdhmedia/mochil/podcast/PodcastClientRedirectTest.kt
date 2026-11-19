package net.kdhmedia.mochil.podcast

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Test

class PodcastClientRedirectTest {
    @Test fun `show catalogue follows safe API redirect while subscription stays POST`() {
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        var mutationMethod = ""
        server.createContext("/api/android/v1/podcast-shows") { exchange ->
            exchange.responseHeaders.add("Location", "/api/android/v1/canonical-podcast-shows")
            exchange.sendResponseHeaders(303, -1); exchange.close()
        }
        server.createContext("/api/android/v1/canonical-podcast-shows") { exchange ->
            val body = """[{"id":"show","title":"Show","episode_count":4}]""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
        }
        server.createContext("/api/android/v1/podcast-feeds") { exchange ->
            mutationMethod = exchange.requestMethod
            exchange.requestBody.use { it.readBytes() }
            exchange.sendResponseHeaders(204, -1); exchange.close()
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val client = PodcastClient()
            assertEquals(listOf("Show"), client.shows(credential, 0, "").items.map(PodcastShow::title))
            client.subscribe(credential, "https://feeds.example/show.xml")
            assertEquals("POST", mutationMethod)
        } finally { server.stop(0) }
    }
}
