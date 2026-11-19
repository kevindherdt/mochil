package net.kdhmedia.mochil.dashboard

import android.content.Context
import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import org.robolectric.annotation.Implementation
import org.robolectric.annotation.Implements

@Implements(DeviceCredentialStore::class)
class QuietRefreshCredentialStoreShadow {
    @Implementation fun __constructor__(context: Context) {}
    @Implementation fun load(): DeviceCredential? = null
}

/**
 * The resume observer refreshes Home every time the app comes back — from the reader, from the player, from another
 * app. Doing that with the loading flag raised is what made the pull-to-refresh indicator spin on its own, so the
 * resume path refreshes without it and only a real pull (or the first load) shows the spinner.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [QuietRefreshCredentialStoreShadow::class])
class DashboardQuietRefreshTest {
    private val home = """{"reading":[{"id":"book","library_id":"books","title":"Book","authors":[]}],"listening":[],"watching":[]}"""

    @Test fun aResumeRefreshKeepsTheSpinnerQuietAndStillUpdatesHome() {
        val calls = AtomicInteger()
        val entered = CountDownLatch(1)
        val release = CountDownLatch(1)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.executor = Executors.newCachedThreadPool { runnable -> Thread(runnable).apply { isDaemon = true } }
        server.createContext("/") { exchange ->
            if (exchange.requestURI.path.endsWith("/home")) {
                // Only the second request blocks, so the test can inspect the state while a resume refresh is in flight.
                if (calls.incrementAndGet() == 2) { entered.countDown(); release.await(10, TimeUnit.SECONDS) }
                val bytes = home.toByteArray()
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            } else {
                val bytes = "[]".toByteArray()
                exchange.sendResponseHeaders(200, bytes.size.toLong())
                exchange.responseBody.use { it.write(bytes) }
            }
        }
        server.start()
        try {
            val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
            DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            fun idleUntil(condition: () -> Boolean) {
                val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
                while (!condition() && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
                shadowOf(Looper.getMainLooper()).idle()
                assertTrue("Timed out waiting for Home", condition())
            }

            // A real load raises the flag: that is the first paint and a pull, the two cases the spinner belongs to.
            vm.refresh(); shadowOf(Looper.getMainLooper()).idle()
            assertTrue("an explicit refresh should show the indicator", vm.state.value.loading)
            idleUntil { !vm.state.value.loading && vm.state.value.reading.isNotEmpty() }

            vm.refreshQuietly(); shadowOf(Looper.getMainLooper()).idle()
            assertTrue("the resume refresh should actually reach the server", entered.await(3, TimeUnit.SECONDS))
            shadowOf(Looper.getMainLooper()).idle()
            assertFalse("a resume must not spin the pull-to-refresh indicator", vm.state.value.loading)
            assertNull(vm.state.value.error)

            release.countDown()
            idleUntil { vm.state.value.reading.isNotEmpty() && !vm.state.value.loading }
            assertEquals("Book", vm.state.value.reading.single().book.title)
        } finally {
            release.countDown()
            server.stop(0)
        }
    }
}
