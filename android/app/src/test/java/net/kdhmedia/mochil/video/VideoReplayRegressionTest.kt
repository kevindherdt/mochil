package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoReplayRegressionTest {
    @Test fun `closing a finished replay at zero does not clear watched status`() {
        var puts = 0
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            if (exchange.requestMethod == "PUT") puts++
            val body = """{"revision":8,"position_seconds":0,"finished":false}""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
            exchange.close()
        }
        server.start()
        try {
            val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "token")
            assertTrue(VideoProgressClient().save(credential, entry, 0.0, 100.0, false))
            assertEquals(0, puts)
        } finally { server.stop(0) }
    }
    @Test fun `finished replay sends guarded restart and does not overwrite a newer finished revision`() {
        for (conflict in listOf(false, true)) {
            val requests = mutableListOf<JSONObject>()
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/video-progress/") { exchange ->
                if (exchange.requestMethod == "PUT") requests.add(JSONObject(exchange.requestBody.readBytes().decodeToString()))
                if (conflict && exchange.requestMethod == "PUT") exchange.sendResponseHeaders(409, -1)
                else {
                    val body = (if (conflict) """{"revision":8,"position_seconds":100,"finished":true}""" else """{"revision":8,"position_seconds":10,"finished":false}""").encodeToByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
                }
                exchange.close()
            }
            server.start()
            try {
                val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "token")
                assertTrue(VideoProgressClient().save(credential, entry, 10.0, 100.0, false))
                assertEquals(1, requests.size)
                assertEquals(7L, requests.single().getLong("expected_revision"))
                assertTrue(requests.single().getBoolean("explicit_seek"))
            } finally { server.stop(0) }
        }
    }

    @Test fun `conflicted restart and later passive samples leave newer unfinished remote alone`() {
        val requests = mutableListOf<JSONObject>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            if (exchange.requestMethod == "PUT") {
                requests.add(JSONObject(exchange.requestBody.readBytes().decodeToString()))
                exchange.sendResponseHeaders(409, -1)
            } else {
                val body = """{"revision":8,"position_seconds":10,"finished":false}""".encodeToByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
            }
            exchange.close()
        }
        server.start()
        try {
            val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "token")
            val client = VideoProgressClient()
            assertTrue(client.save(credential, entry, 30.0, 100.0, false))
            assertTrue(client.save(credential, entry, 40.0, 100.0, false))
            assertEquals(1, requests.size)
            assertEquals(7L, requests.single().getLong("expected_revision"))
        } finally { server.stop(0) }
    }

    @Test fun `older journal conflict cannot let a finished local replay claim newer remote`() {
        val requests = mutableListOf<JSONObject>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            if (exchange.requestMethod == "PUT") {
                requests.add(JSONObject(exchange.requestBody.readBytes().decodeToString()))
                exchange.sendResponseHeaders(409, -1)
            } else {
                val body = """{"revision":8,"position_seconds":10,"finished":false}""".encodeToByteArray()
                exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
            }
            exchange.close()
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "token")
            val client = VideoProgressClient()
            val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
            assertTrue(client.save(credential, entry.copy(positionMs = 0, finished = false), 5.0, 100.0, false))
            assertTrue(client.save(credential, entry, 30.0, 100.0, false))
            assertEquals(1, requests.size)
        } finally { server.stop(0) }
    }

    @Test fun `pending explicit seek restores start without erasing remote snapshot`() {
        val owner = "a".repeat(64)
        val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
        val pending = PendingVideoProgress(owner, "lib", "item", 20.0, 100.0, 7, false, 1000)
        val queue = restorePendingVideoPosition(VideoPlaybackQueue(listOf(entry), 0), listOf(pending), owner, 2000)
        assertEquals(20000L, queue.startPositionMs())
        assertTrue(queue.entries.single().finished)
        assertEquals(100000L, queue.entries.single().positionMs)
    }

    @Test fun `offline replay resumes only against its original finished revision`() {
        val owner = "a".repeat(64)
        val entry = VideoPlaybackEntry("lib", "item", "", "url", "", 100000, 100.0, "", progressRevision = 7, finished = true)
        val pending = PendingVideoProgress(owner, "lib", "item", 20.0, 100.0, 7, false, replayFinished = true)
        val saved = decodePendingVideoProgressStrict(encodePendingVideoProgress(listOf(pending)))
        assertEquals(20000L, restorePendingVideoPosition(VideoPlaybackQueue(listOf(entry), 0), saved, owner, 2000).startPositionMs())
        assertEquals(0L, restorePendingVideoPosition(VideoPlaybackQueue(listOf(entry.copy(progressRevision = 8)), 0), saved, owner, 2000).startPositionMs())
    }
}
