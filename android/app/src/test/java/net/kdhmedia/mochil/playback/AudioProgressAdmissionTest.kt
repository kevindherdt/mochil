package net.kdhmedia.mochil.playback

import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicReference
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AudioProgressAdmissionTest {
    @Test fun `admission is checked after waiting behind a slow request lock`() {
        val readStarted = CountDownLatch(1)
        val releaseRead = CountDownLatch(1)
        val waiterStarted = CountDownLatch(1)
        val waiterThread = AtomicReference<Thread>()
        val allowed = AtomicBoolean(true)
        val requests = AtomicInteger()
        val executor = Executors.newFixedThreadPool(2)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/api/android/v1/audio-progress/") { exchange ->
            requests.incrementAndGet()
            readStarted.countDown()
            releaseRead.await(5, TimeUnit.SECONDS)
            val body = """{"revision":1,"position_seconds":200,"finished":false}""".encodeToByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
            val client = AudioProgressClient()
            val first = executor.submit<Boolean> { client.save(credential, "podcasts", "first", 30.0, 600.0) }
            assertTrue(readStarted.await(2, TimeUnit.SECONDS))
            val waiting = executor.submit<Boolean> {
                waiterThread.set(Thread.currentThread())
                waiterStarted.countDown()
                client.save(credential, "podcasts", "waiting", 30.0, 600.0, maySend = allowed::get)
            }
            assertTrue(waiterStarted.await(2, TimeUnit.SECONDS))
            val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(2)
            while (waiterThread.get().state != Thread.State.BLOCKED && System.nanoTime() < deadline) Thread.yield()
            assertEquals(Thread.State.BLOCKED, waiterThread.get().state)
            allowed.set(false)
            releaseRead.countDown()
            assertTrue(first.get(2, TimeUnit.SECONDS))
            assertFalse(waiting.get(2, TimeUnit.SECONDS))
            assertEquals(1, requests.get())
        } finally {
            releaseRead.countDown()
            executor.shutdownNow()
            server.stop(0)
            executor.awaitTermination(2, TimeUnit.SECONDS)
        }
    }

    @Test fun `superseded send stops before initial read mutation or conflict retry`() {
        for (stage in listOf("before", "get", "conflict")) {
            val allowed = AtomicBoolean(stage != "before")
            val requests = AtomicInteger()
            val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
            server.createContext("/api/android/v1/audio-progress/podcasts/one") { exchange ->
                requests.incrementAndGet()
                exchange.requestBody.use { it.readBytes() }
                if (exchange.requestMethod == "PUT") {
                    allowed.set(false)
                    exchange.sendResponseHeaders(409, -1)
                    exchange.close()
                } else {
                    if (stage == "get") allowed.set(false)
                    val body = """{"revision":0,"position_seconds":0,"finished":false}""".encodeToByteArray()
                    exchange.sendResponseHeaders(200, body.size.toLong())
                    exchange.responseBody.use { it.write(body) }
                }
            }
            server.start()
            try {
                val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
                assertFalse(AudioProgressClient().save(credential, "podcasts", "one", 30.0, 600.0, maySend = allowed::get))
                assertEquals(when (stage) { "before" -> 0; "get" -> 1; else -> 2 }, requests.get())
            } finally { server.stop(0) }
        }
    }
}
