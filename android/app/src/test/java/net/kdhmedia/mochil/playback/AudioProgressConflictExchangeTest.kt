package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class AudioProgressConflictExchangeTest {
    @Test fun `newer remote position wins after conflict without a second write`() {
        withServer(remoteAfterConflict = 240.0) { client, credential, requests, writes ->
            assertTrue(client.save(credential, "podcasts", "episode", 120.0, 600.0))
            assertEquals(listOf("GET", "PUT", "GET"), requests)
            assertEquals(listOf(1L), writes.map { it.getLong("expected_revision") })
        }
    }

    @Test fun `still newer local progress retries once with the refreshed revision`() {
        withServer(remoteAfterConflict = 60.0) { client, credential, requests, writes ->
            assertTrue(client.save(credential, "podcasts", "episode", 120.0, 600.0))
            assertEquals(listOf("GET", "PUT", "GET", "PUT"), requests)
            assertEquals(listOf(1L, 2L), writes.map { it.getLong("expected_revision") })
            assertTrue(writes.all { !it.getBoolean("explicit_seek") })
            assertEquals(120.0, writes.last().getDouble("position_seconds"), 0.0)
        }
    }

    @Test fun `repeated conflict stops retrying and discards cached revision`() {
        withServer(remoteAfterConflict = 60.0, rejectRetry = true) { client, credential, requests, _ ->
            assertFalse(client.save(credential, "podcasts", "episode", 120.0, 600.0))
            assertEquals(listOf("GET", "PUT", "GET", "PUT"), requests)
            // A later delivery must fetch again, not trust the contested revision.
            assertFalse(client.save(credential, "podcasts", "episode", 130.0, 600.0))
            assertEquals("GET", requests[4])
            assertEquals(8, requests.size)
        }
    }

    private fun withServer(
        remoteAfterConflict: Double,
        rejectRetry: Boolean = false,
        test: (AudioProgressClient, DeviceCredential, List<String>, List<JSONObject>) -> Unit,
    ) {
        val requests = mutableListOf<String>()
        val writes = mutableListOf<JSONObject>()
        var reads = 0
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/audio-progress/podcasts/episode") { exchange ->
            requests.add(exchange.requestMethod)
            val body = if (exchange.requestMethod == "GET") {
                reads++
                """{"revision":${if (reads == 1) 1 else 2},"position_seconds":${if (reads == 1) 0.0 else remoteAfterConflict},"finished":false}"""
            } else {
                writes.add(JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() }))
                if (writes.size == 1 || rejectRetry) {
                    exchange.sendResponseHeaders(409, -1)
                    exchange.close()
                    return@createContext
                }
                """{"revision":3,"position_seconds":120,"finished":false}"""
            }.encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            test(AudioProgressClient(), DeviceCredential("http://127.0.0.1:${server.address.port}", "test-device", "test-token"), requests, writes)
        } finally { server.stop(0) }
    }
}
