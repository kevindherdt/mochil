package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicBoolean
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

/** Real HTTP replay plus serialized journal recreation, without an Android service mock. */
class AudioProgressDurableRecoveryTest {
    @Test fun `departing and current episode checkpoints survive failed delivery and recreation`() = runBlocking {
        val offline = AtomicBoolean(true)
        val received = mutableMapOf<String, Double>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/audio-progress/") { exchange ->
            try {
                if (offline.get()) {
                    exchange.sendResponseHeaders(503, -1)
                } else {
                    val id = exchange.requestURI.path.substringAfterLast('/')
                    var position = 0.0
                    var revision = 0
                    if (exchange.requestMethod == "PUT") {
                        val body = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                        position = body.getDouble("position_seconds")
                        received[id] = position
                        revision = 1
                    }
                    val body = """{"revision":$revision,"position_seconds":$position,"finished":false}""".encodeToByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong())
                    exchange.responseBody.use { it.write(body) }
                }
            } finally { exchange.close() }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "fixture-token")
            var stored: ByteArray? = null
            fun journal() = AudioProgressJournal({ stored }, { stored = it; true }, { stored = null; true })
            val first = journal()
            val owner = audioProgressOwnerIdentity(credential.deviceId)
            val departing = PendingAudioProgress("podcasts", "departing", 590.0, 600.0, owner = owner)
            val current = departing.copy(itemId = "current", positionSeconds = 42.0)
            assertNotNull(first.put(departing))
            assertNotNull(first.put(current))
            suspend fun replay(outbox: AudioProgressJournal) {
                val client = AudioProgressClient()
                replayPendingAudioProgress(outbox.entries(credential.deviceId),
                    send = { client.save(credential, it.libraryId, it.itemId, it.positionSeconds, it.durationSeconds) },
                    acknowledge = outbox::removeIfCurrent)
            }
            replay(first)
            assertEquals(2, first.entries(credential.deviceId).size)
            assertTrue(received.isEmpty())
            offline.set(false)
            val recreated = journal()
            replay(recreated)
            assertEquals(mapOf("departing" to 590.0, "current" to 42.0), received)
            assertTrue(recreated.entries(credential.deviceId).isEmpty())
            assertNull(stored)
        } finally { server.stop(0) }
    }
}
