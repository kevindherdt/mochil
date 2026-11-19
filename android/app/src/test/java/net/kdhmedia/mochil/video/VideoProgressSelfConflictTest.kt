package net.kdhmedia.mochil.video

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.*
import net.kdhmedia.mochil.auth.DeviceCredential
import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class VideoProgressSelfConflictTest {
    @Test fun `acknowledgment without pending entry survives recreation and does not regress`() {
        val owner = "a".repeat(64)
        var bytes: ByteArray? = null
        fun journal() = VideoProgressJournal(owner, { bytes }, { bytes = it; true })
        val saved = PendingVideoProgress(owner, "library", "item", 12.0, 600.0, 7, false, mediaRevision = "file", replayFinished = true)
        val j = journal()
        assertEquals(saved, j.put(saved))
        assertTrue(j.removeIfCurrent(saved))
        assertTrue(j.rebaseNewer(saved, 8))
        val next = journal().put(saved.copy(positionSeconds = 30.0))!!
        assertEquals(8L, next.revision)
        assertTrue(journal().rebaseNewer(next, 9))
        assertTrue(journal().rebaseNewer(saved, 8))
        assertEquals(9L, journal().put(saved.copy(positionSeconds = 40.0))!!.revision)
    }
    @Test fun `consecutive in flight saves rebase using actual sent revision`() = runBlocking {
        val received = Array(3) { CountDownLatch(1) }
        val release = Array(2) { CountDownLatch(1) }
        val calls = AtomicInteger()
        val expected = java.util.Collections.synchronizedList(mutableListOf<Long>())
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                val body = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                expected.add(body.getLong("expected_revision"))
                val index = calls.getAndIncrement()
                received[index].countDown()
                if (index < 2) {
                    check(release[index].await(5, TimeUnit.SECONDS))
                    val response = """{"revision":${8 + index},"position_seconds":${12 + index * 18},"finished":false}""".toByteArray()
                    exchange.sendResponseHeaders(200, response.size.toLong())
                    exchange.responseBody.use { it.write(response) }
                } else exchange.sendResponseHeaders(503, -1)
            } finally { exchange.close() }
        }
        server.start()
        val job = SupervisorJob(); val scope = CoroutineScope(job + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            val journal = VideoProgressJournal(videoProgressOwner(credential), { bytes }, { bytes = it; true })
            val reporter = VideoProgressReporter(scope, { 100 }, { 1000 }, journal)
            val entry = VideoPlaybackEntry("anime", "one", "", "", "", 600_000, 600.0, "", revision = "file-r1", progressRevision = 7, finished = true)
            reporter.submit(credential, entry, 12_000, 600_000, false, false)
            assertTrue(received[0].await(5, TimeUnit.SECONDS))
            reporter.submit(credential, entry, 30_000, 600_000, false, false)
            release[0].countDown()
            assertTrue(received[1].await(5, TimeUnit.SECONDS))
            reporter.submit(credential, entry, 40_000, 600_000, false, false)
            assertEquals(8L, journal.entries().single().revision)
            release[1].countDown()
            assertTrue(received[2].await(5, TimeUnit.SECONDS))
            assertEquals(listOf(7L, 8L, 9L), expected.toList())
            assertEquals(9L, journal.entries().single().revision)
        } finally {
            release.forEach { it.countDown() }; scope.cancel(); server.stop(0); job.join()
        }
    }

    @Test fun `saved restart rebases later offline progress across recreation`() = exercise(false)
    @Test fun `rebased restart does not overwrite concurrent remote progress`() = exercise(true)
    @Test fun `empty journal acknowledgment rebases later replay`() = exercise(false, true)
    @Test fun `empty journal acknowledgment still detects remote advancement`() = exercise(true, true)

    private fun exercise(remoteAdvanced: Boolean, emptyAtAcknowledgment: Boolean = false) = runBlocking {
        val firstReceived = CountDownLatch(1)
        val release = CountDownLatch(1)
        val secondReceived = CountDownLatch(1)
        val thirdReceived = CountDownLatch(1)
        val calls = AtomicInteger()
        var remoteRevision = 7L
        var remotePosition = 600.0
        var remoteFinished = true
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/video-progress/") { exchange ->
            try {
                if (exchange.requestMethod == "GET") {
                    val body = """{"revision":$remoteRevision,"position_seconds":$remotePosition,"finished":$remoteFinished}""".toByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong()); exchange.responseBody.use { it.write(body) }
                } else {
                    val body = JSONObject(exchange.requestBody.use { it.readBytes().decodeToString() })
                    when (calls.incrementAndGet()) {
                        1 -> {
                            firstReceived.countDown(); release.await(3, TimeUnit.SECONDS)
                            remoteRevision = 8; remotePosition = 12.0; remoteFinished = false
                            val response = """{"revision":8,"position_seconds":12,"finished":false}""".toByteArray()
                            exchange.sendResponseHeaders(200, response.size.toLong()); exchange.responseBody.use { it.write(response) }
                        }
                        2 -> { exchange.sendResponseHeaders(503, -1); secondReceived.countDown() }
                        3 -> { exchange.sendResponseHeaders(503, -1); thirdReceived.countDown() }
                        else -> {
                            if (body.getLong("expected_revision") != remoteRevision) exchange.sendResponseHeaders(409, -1)
                            else {
                                remoteRevision++; remotePosition = body.getDouble("position_seconds")
                                val response = """{"revision":$remoteRevision,"position_seconds":$remotePosition,"finished":false}""".toByteArray()
                                exchange.sendResponseHeaders(200, response.size.toLong()); exchange.responseBody.use { it.write(response) }
                            }
                        }
                    }
                }
            } finally { exchange.close() }
        }
        server.start()
        val job = SupervisorJob(); val scope = CoroutineScope(job + Dispatchers.Default)
        val restartedJob = SupervisorJob(); val restarted = CoroutineScope(restartedJob + Dispatchers.Default)
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "tv", "fixture")
            var bytes: ByteArray? = null
            fun journal() = VideoProgressJournal(videoProgressOwner(credential), { bytes }, { bytes = it; true })
            val outbox = journal()
            val reporter = VideoProgressReporter(scope, { 100 }, { 1000 }, outbox)
            val entry = VideoPlaybackEntry("anime", "one", "", "", "", 600_000, 600.0, "", revision = "file-r1", progressRevision = 7, finished = true)
            reporter.submit(credential, entry, 12_000, 600_000, false, false)
            assertTrue(firstReceived.await(3, TimeUnit.SECONDS))
            if (!emptyAtAcknowledgment) reporter.submit(credential, entry, 30_000, 600_000, false, false)
            release.countDown()
            if (emptyAtAcknowledgment) {
                withTimeout(3000) { while (journal().entries().isNotEmpty()) delay(10) }
                reporter.submit(credential, entry, 30_000, 600_000, false, false)
            }
            assertTrue(secondReceived.await(3, TimeUnit.SECONDS))
            reporter.submit(credential, entry, 40_000, 600_000, false, false)
            assertEquals(8L, journal().entries().single().revision)
            assertTrue(thirdReceived.await(3, TimeUnit.SECONDS))
            scope.cancel(); job.join()
            assertEquals(8L, journal().entries().single().revision)
            if (remoteAdvanced) { remoteRevision = 9; remotePosition = 100.0 }
            VideoProgressReporter(restarted, { 500 }, { 700_000 }, journal()).replay(credential).join()
            withTimeout(3000) { while (journal().entries().isNotEmpty()) delay(10) }
            assertEquals(if (remoteAdvanced) 100.0 else 40.0, remotePosition, 0.0)
        } finally { release.countDown(); scope.cancel(); restarted.cancel(); server.stop(0); job.join(); restartedJob.join() }
    }
}
