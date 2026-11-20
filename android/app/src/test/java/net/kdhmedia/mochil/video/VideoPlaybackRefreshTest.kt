package net.kdhmedia.mochil.video

import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.ShadowDeviceCredentialStore
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [ShadowDeviceCredentialStore::class])
class VideoPlaybackRefreshTest {
    @Test fun returningFromPlaybackReloadsSelectedSeasonProgress() {
        val position = AtomicInteger(20)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            val body = (if (exchange.requestURI.path.endsWith("/videos")) """[{"id":"ep","library_id":"tv","title":"Pilot","show":"Show","season":2,"episode":1,"stream_url":"/stream","progress":{"position_seconds":${position.get()}}}]""" else "[]").toByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
        }
        server.start()
        try {
            val vm = VideoViewModel(RuntimeEnvironment.getApplication())
            VideoViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            vm.open("tv", "tv")
            vm.openShow(VideoShow("Show", "Show", "show", 2, listOf(1, 2), ""), 2)
            fun awaitPosition(expected: Double) {
                val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
                while (vm.state.value.items.firstOrNull()?.resumeSeconds != expected && System.nanoTime() < deadline) {
                    shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10)
                }
                assertEquals(expected, vm.state.value.items.firstOrNull()?.resumeSeconds ?: -1.0, 0.0)
            }
            awaitPosition(20.0)
            position.set(120)
            vm.refreshAfterPlayback()
            awaitPosition(120.0)
            assertEquals(2, vm.state.value.selectedSeason)
        } finally { server.stop(0) }
    }
}
