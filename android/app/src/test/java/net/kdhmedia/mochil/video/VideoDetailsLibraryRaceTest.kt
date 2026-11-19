package net.kdhmedia.mochil.video

import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.ShadowDeviceCredentialStore
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [ShadowDeviceCredentialStore::class])
class VideoDetailsLibraryRaceTest {
    @Test fun delayedOldLibraryDetailsCannotReplaceNewLibraryDetails() {
        val started = CountDownLatch(1)
        val release = CountDownLatch(1)
        val returned = CountDownLatch(1)
        val pool = Executors.newCachedThreadPool()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.executor = pool
        server.createContext("/") { exchange ->
            val path = exchange.requestURI.path
            val body = when {
                path.endsWith("/video-details/old") -> { started.countDown(); release.await(5, TimeUnit.SECONDS); "{\"certification\":\"OLD\"}" }
                path.endsWith("/video-details/new") -> "{\"certification\":\"NEW\"}"
                else -> "[]"
            }.toByteArray()
            exchange.sendResponseHeaders(200, body.size.toLong())
            exchange.responseBody.use { it.write(body) }
            if (path.endsWith("/video-details/old")) returned.countDown()
        }
        server.start()
        try {
            val vm = VideoViewModel(RuntimeEnvironment.getApplication())
            VideoViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            val item = VideoItem(id = "same", libraryId = "old", title = "Movie", show = "", season = 0, episode = 0, durationSeconds = 0.0, resumeSeconds = 0.0, artworkPath = "", streamPath = "", hlsPath = "", subtitles = emptyList())
            vm.open("old", "movies")
            vm.loadDetails(null, item)
            assertTrue(started.await(5, TimeUnit.SECONDS))
            vm.open("new", "movies")
            vm.loadDetails(null, item.copy(libraryId = "new"))
            val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
            while (vm.state.value.details["item:same"]?.certification != "NEW" && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
            assertEquals("NEW", vm.state.value.details["item:same"]?.certification)
            release.countDown()
            assertTrue(returned.await(5, TimeUnit.SECONDS))
            val until = System.nanoTime() + TimeUnit.MILLISECONDS.toNanos(500)
            while (System.nanoTime() < until) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
            assertEquals("NEW", vm.state.value.details["item:same"]?.certification)
        } finally { release.countDown(); server.stop(0); pool.shutdownNow() }
    }
}
