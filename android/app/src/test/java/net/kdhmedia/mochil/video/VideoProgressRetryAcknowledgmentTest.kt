package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressRetryAcknowledgmentTest {
    @Test fun `rebased checkpoint cannot overwrite newer remote progress after recreation`() {
        val owner = "a".repeat(64)
        var bytes: ByteArray? = null
        fun journal() = VideoProgressJournal(owner, { bytes }, { bytes = it; true })
        val old = PendingVideoProgress(owner, "lib", "item", 30.0, 100.0, 7, false)
        journal().put(old.copy(positionSeconds = 40.0))
        assertTrue(journal().rebaseNewer(old.copy(revision = 8, rebasedFromRevision = 7), 9, 7))
        val pending = journal().entries().single()
        assertEquals(9L, pending.revision)
        val requests = mutableListOf<Long>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                if (exchange.requestMethod == "PUT") {
                    requests.add(JSONObject(exchange.requestBody.readBytes().decodeToString()).getLong("expected_revision"))
                    exchange.sendResponseHeaders(409, -1)
                } else {
                    val body = """{"revision":10,"position_seconds":80,"finished":false}""".toByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong())
                    exchange.responseBody.use { it.write(body) }
                }
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            val entry = VideoPlaybackEntry("lib", "item", "", "", "", 0, 100.0, "", progressRevision = pending.revision)
            assertTrue(VideoProgressClient().save(credential, entry, pending.positionSeconds, pending.durationSeconds, false))
            assertEquals(listOf(9L), requests)
        } finally { server.stop(0) }
    }

    @Test fun `retry acknowledgment rebases an in flight newer checkpoint`() {
        val owner = "a".repeat(64)
        var bytes: ByteArray? = null
        fun journal() = VideoProgressJournal(owner, { bytes }, { bytes = it; true })
        val old = PendingVideoProgress(owner, "lib", "item", 30.0, 100.0, 7, false)
        val newer = old.copy(positionSeconds = 40.0)
        assertNotNull(journal().put(old))
        assertNotNull(journal().put(newer))
        assertTrue(journal().rebaseNewer(old.copy(revision = 8, rebasedFromRevision = 7), 9, old.revision))
        assertEquals(9L, journal().entries().single().revision)
    }

    @Test fun `successful conflict retry acknowledges actual sent revision`() {
        val requests = mutableListOf<Long>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                if (exchange.requestMethod == "GET") {
                    val response = """{"revision":8,"position_seconds":10,"finished":false}""".toByteArray()
                    exchange.sendResponseHeaders(200, response.size.toLong())
                    exchange.responseBody.use { it.write(response) }
                } else {
                    requests.add(JSONObject(exchange.requestBody.readBytes().decodeToString()).getLong("expected_revision"))
                    if (requests.size == 1) exchange.sendResponseHeaders(409, -1)
                    else {
                        val response = """{"revision":9,"position_seconds":30,"finished":false}""".toByteArray()
                        exchange.sendResponseHeaders(200, response.size.toLong())
                        exchange.responseBody.use { it.write(response) }
                    }
                }
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            val entry = VideoPlaybackEntry("lib", "item", "", "", "", 10000, 100.0, "", progressRevision = 7)
            val acknowledgments = mutableListOf<Pair<Long, Long>>()
            assertTrue(VideoProgressClient().save(credential, entry, 30.0, 100.0, false, onSaved = { sent, saved -> acknowledgments.add(sent to saved) }))
            assertEquals(listOf(7L, 8L), requests)
            assertEquals(listOf(8L to 9L), acknowledgments)
        } finally { server.stop(0) }
    }
}
