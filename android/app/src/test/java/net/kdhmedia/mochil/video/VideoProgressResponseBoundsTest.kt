package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.io.IOException
import java.net.InetSocketAddress
import java.util.Collections
import java.util.concurrent.atomic.AtomicBoolean
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.playback.MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressResponseBoundsTest {
    @Test fun `oversized save or conflict read cannot replace cached revision or trigger another upload`() {
        for (stage in listOf("PUT", "GET")) {
            val oversized = AtomicBoolean(true)
            val methods = Collections.synchronizedList(mutableListOf<String>())
            val revisions = Collections.synchronizedList(mutableListOf<Long>())
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/video-progress/") { exchange ->
                val request = exchange.requestBody.use { it.readBytes().decodeToString() }
                methods += exchange.requestMethod
                if (exchange.requestMethod == "PUT") revisions += JSONObject(request).getLong("expected_revision")
                try {
                    if (oversized.get() && stage == "GET" && exchange.requestMethod == "PUT") {
                        exchange.sendResponseHeaders(409, -1)
                    } else {
                        val json = if (oversized.get()) {
                            """{"revision":99,"position_seconds":999,"finished":true}"""
                        } else """{"revision":2,"position_seconds":30,"finished":false}"""
                        val body = json.encodeToByteArray() + if (oversized.get()) ByteArray(MAX_PLAYBACK_PROGRESS_RESPONSE_BYTES) { 32 } else byteArrayOf()
                        exchange.sendResponseHeaders(200, body.size.toLong())
                        exchange.responseBody.use { it.write(body) }
                    }
                } finally { exchange.close() }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
                val entry = VideoPlaybackEntry("anime", "episode", "Episode", "unused", "", 0, 600.0, "")
                val client = VideoProgressClient()
                assertThrows(IOException::class.java) { client.save(credential, entry, 30.0, 600.0, false) }
                assertEquals(if (stage == "GET") listOf("PUT", "GET") else listOf("PUT"), methods.toList())
                oversized.set(false)
                client.save(credential, entry, 30.0, 600.0, false)
                client.save(credential, entry, 40.0, 600.0, false)
                assertEquals(listOf(0L, 0L, 2L), revisions.toList())
            } finally { server.stop(0) }
        }
    }
}
