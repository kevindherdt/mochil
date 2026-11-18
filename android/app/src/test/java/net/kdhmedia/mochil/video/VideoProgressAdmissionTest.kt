package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.Collections
import java.util.concurrent.atomic.AtomicBoolean
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.*
import org.junit.Test

class VideoProgressAdmissionTest {
    @Test fun `superseded progress stops before sending or continuing conflict recovery`() {
        for (stage in listOf("before", "conflict", "get")) {
            val allowed = AtomicBoolean(stage != "before")
            val methods = Collections.synchronizedList(mutableListOf<String>())
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/video-progress/") { exchange ->
                methods += exchange.requestMethod
                exchange.requestBody.use { it.readBytes() }
                try {
                    if (exchange.requestMethod == "PUT") {
                        if (stage == "conflict") allowed.set(false)
                        exchange.sendResponseHeaders(409, -1)
                    } else {
                        allowed.set(false)
                        val body = """{"revision":2,"position_seconds":0,"finished":false}""".encodeToByteArray()
                        exchange.sendResponseHeaders(200, body.size.toLong())
                        exchange.responseBody.use { it.write(body) }
                    }
                } finally { exchange.close() }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
                val entry = VideoPlaybackEntry("anime", "episode", "Episode", "unused", "", 0, 600.0, "")
                assertFalse(VideoProgressClient().save(credential, entry, 30.0, 600.0, false, maySend = allowed::get))
                assertEquals(when (stage) { "before" -> emptyList(); "conflict" -> listOf("PUT"); else -> listOf("PUT", "GET") }, methods.toList())
            } finally { server.stop(0) }
        }
    }
}
