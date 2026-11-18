package net.kdhmedia.mochil.catalogue

import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.flow.MutableStateFlow
import net.kdhmedia.mochil.auth.DeviceCredential
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import org.robolectric.shadows.ShadowSystemClock
import org.robolectric.annotation.Implements
import org.robolectric.annotation.Implementation
import android.content.Context
import net.kdhmedia.mochil.auth.DeviceCredentialStore

import java.time.Duration

@Implements(DeviceCredentialStore::class)
class ShadowDeviceCredentialStore {
    @Implementation fun __constructor__(context: Context) {}
    @Implementation fun load(): DeviceCredential? = null
}

@Implements(LibraryVisibilityStore::class)
class ShadowLibraryVisibilityStore {
    @Implementation fun __constructor__(context: Context) {}
    @Implementation fun filter(libraries: List<LibrarySummary>) = libraries
    @Implementation fun getDeviceClass() = ClientDeviceClass.Handheld
}

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [ShadowDeviceCredentialStore::class, ShadowLibraryVisibilityStore::class])
class LibraryOfflineResumeTest {
    @Test fun resumeReloadsTheSameShelfAfterOfflineMode() = checkRecovery(userSelectedDownloads = false)

    @Test fun resumePreservesDownloadsOnlySelectedWhileOffline() = checkRecovery(userSelectedDownloads = true)

    @Test fun resumePreservesDownloadsOnlySelectedBeforeOffline() = checkRecovery(userSelectedDownloads = true, selectBeforeOffline = true)

    @Test fun retryAfterFailedForegroundRefreshReconnectsImmediately() {
        val paths = CopyOnWriteArrayList<String>()
        val online = java.util.concurrent.atomic.AtomicBoolean(false)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            paths.add(exchange.requestURI.path)
            val body = if (exchange.requestURI.path.endsWith("/libraries"))
                """[{"id":"books","name":"Books","kind":"books","count":1}]""" else "[]"
            val bytes = body.toByteArray()
            exchange.sendResponseHeaders(if (online.get()) 200 else 503, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val vm = LibraryViewModel(RuntimeEnvironment.getApplication())
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            val library = LibrarySummary("books", "Books", "books", 1)
            @Suppress("UNCHECKED_CAST")
            val state = LibraryViewModel::class.java.getDeclaredField("mutableState").apply { isAccessible = true }
                .get(vm) as MutableStateFlow<LibraryState>
            state.value = LibraryState(allLibraries = listOf(library), libraries = listOf(library), selected = library,
                initialLoading = false, offlineOnly = true, downloadedOnly = true, grouping = PublicationGrouping.Books)
            ShadowSystemClock.advanceBy(Duration.ofSeconds(20))
            vm.refreshLibrarySummaries()
            val deadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < deadline && paths.none { it.endsWith("/libraries") }) {
                shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(20)
            }
            assertTrue(paths.any { it.endsWith("/libraries") })
            Thread.sleep(100); shadowOf(Looper.getMainLooper()).idle()
            online.set(true)
            vm.retry()
            val retryDeadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < retryDeadline && vm.state.value.offlineOnly) {
                shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(20)
            }
            assertFalse("Retry must reconnect after a failed foreground refresh; requests were $paths", vm.state.value.offlineOnly)
        } finally { server.stop(0) }
    }

    @Test fun cachedFallbackShowsSafeHttpDiagnostic() {
        val secret = "private-token-and-library-name"
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            val bytes = secret.toByteArray()
            exchange.sendResponseHeaders(503, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val app = RuntimeEnvironment.getApplication()
            val library = LibrarySummary("books", "Books", "books", 1)
            LibraryNavigationStore(app).apply { saveLibraries(listOf(library)); select("books") }
            val vm = LibraryViewModel(app)
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            vm.refresh()
            val deadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < deadline && !vm.state.value.offlineOnly) {
                shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10)
            }
            assertTrue("Cached libraries should remain usable", vm.state.value.offlineOnly)
            val diagnostic = runCatching {
                vm.state.value.javaClass.getMethod("getOfflineDiagnostic").invoke(vm.state.value) as? String
            }.getOrNull()
            assertTrue("Offline fallback must show the HTTP status", diagnostic?.contains("503") == true)
            assertFalse("Server response bodies must not be displayed", diagnostic.orEmpty().contains(secret))
            assertFalse("Server addresses must not be displayed", diagnostic.orEmpty().contains("127.0.0.1"))
        } finally { server.stop(0) }
    }

    @Test fun explicitRetryDuringPendingForegroundRefreshStartsAnotherAttempt() {
        val arrived = java.util.concurrent.CountDownLatch(1)
        val release = java.util.concurrent.CountDownLatch(1)
        val requests = java.util.concurrent.atomic.AtomicInteger()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            val count = requests.incrementAndGet()
            if (count == 1) { arrived.countDown(); release.await(5, java.util.concurrent.TimeUnit.SECONDS) }
            val bytes = "[]".toByteArray()
            exchange.sendResponseHeaders(if (count == 1) 503 else 200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val vm = LibraryViewModel(RuntimeEnvironment.getApplication())
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            val library = LibrarySummary("books", "Books", "books", 1)
            @Suppress("UNCHECKED_CAST")
            val state = LibraryViewModel::class.java.getDeclaredField("mutableState").apply { isAccessible = true }
                .get(vm) as MutableStateFlow<LibraryState>
            state.value = LibraryState(allLibraries = listOf(library), libraries = listOf(library), selected = library,
                initialLoading = false, offlineOnly = true)
            ShadowSystemClock.advanceBy(Duration.ofSeconds(20))
            vm.refreshLibrarySummaries()
            val deadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < deadline && arrived.count > 0) {
                shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10)
            }
            assertTrue("First request must reach the server", arrived.count == 0L)
            vm.retry()
            release.countDown()
            val retryDeadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < retryDeadline && requests.get() < 2) {
                shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10)
            }
            assertTrue("Explicit retry must not be dropped during an in-flight request", requests.get() >= 2)
        } finally { release.countDown(); server.stop(0) }
    }

    @Test fun foregroundRecoveryLoadsLibrarySelectedWhileRequestWasPending() {
        val arrived = java.util.concurrent.CountDownLatch(1)
        val release = java.util.concurrent.CountDownLatch(1)
        val paths = CopyOnWriteArrayList<String>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            paths.add(exchange.requestURI.toString())
            if (exchange.requestURI.path.endsWith("/libraries")) {
                arrived.countDown(); release.await(5, java.util.concurrent.TimeUnit.SECONDS)
            }
            val body = if (exchange.requestURI.path.endsWith("/libraries"))
                """[{"id":"a","name":"A","kind":"books","count":1},{"id":"b","name":"B","kind":"books","count":1}]""" else "[]"
            val bytes = body.toByteArray(); exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val vm = LibraryViewModel(RuntimeEnvironment.getApplication())
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            val a = LibrarySummary("a", "A", "books", 1); val b = LibrarySummary("b", "B", "books", 1)
            @Suppress("UNCHECKED_CAST")
            val state = LibraryViewModel::class.java.getDeclaredField("mutableState").apply { isAccessible = true }.get(vm) as MutableStateFlow<LibraryState>
            state.value = LibraryState(allLibraries = listOf(a, b), libraries = listOf(a, b), selected = a,
                initialLoading = false, offlineOnly = true, grouping = PublicationGrouping.Books)
            ShadowSystemClock.advanceBy(Duration.ofSeconds(20)); vm.refreshLibrarySummaries()
            val deadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < deadline && arrived.count > 0) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(20) }
            assertTrue(arrived.count == 0L)
            vm.select(b); release.countDown()
            val end = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < end && paths.none { it.contains("/books?") && it.contains("library=b") }) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(20) }
            org.junit.Assert.assertEquals("b", vm.state.value.selected?.id)
            assertTrue("Current selection must load after reconnection; requests were $paths", paths.any { it.contains("/books?") && it.contains("library=b") })
        } finally { release.countDown(); server.stop(0) }
    }

    @Test fun offlineFallbackRestoresSavedGroupingOnReconnect() {
        val online = java.util.concurrent.atomic.AtomicBoolean(false)
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            val body = if (exchange.requestURI.path.endsWith("/libraries"))
                """[{"id":"books","name":"Books","kind":"books","count":1}]""" else "[]"
            val bytes = body.toByteArray(); exchange.sendResponseHeaders(if (online.get()) 200 else 503, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            val app = RuntimeEnvironment.getApplication()
            val library = LibrarySummary("books", "Books", "books", 1)
            LibraryNavigationStore(app).apply { saveLibraries(listOf(library)); select("books"); setGrouping(PublicationGrouping.Authors) }
            val file = java.io.File(app.cacheDir, "offline-test.epub").apply { writeText("test") }
            PublicationDownloadStore(app).record(DownloadedPublication("books", "one", "One", "", "epub", "application/epub+zip", file.absolutePath, file.length()).asBookSummary(), file)
            val vm = LibraryViewModel(app)
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            vm.refresh()
            val offlineDeadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < offlineDeadline && !vm.state.value.offlineOnly) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
            assertTrue(vm.state.value.offlineOnly)
            org.junit.Assert.assertEquals(PublicationGrouping.Books, vm.state.value.grouping)
            assertTrue(vm.state.value.downloadedOnly)
            online.set(true)
            vm.retry()
            val onlineDeadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < onlineDeadline && vm.state.value.offlineOnly) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
            assertFalse(vm.state.value.offlineOnly)
            org.junit.Assert.assertEquals(PublicationGrouping.Authors, vm.state.value.grouping)
            assertFalse(vm.state.value.downloadedOnly)
        } finally { server.stop(0) }
    }

    @Test fun foregroundReconnectsWhileOfflineShelfIsLoading() = checkRecovery(userSelectedDownloads = false, initiallyLoading = true)

    private fun checkRecovery(userSelectedDownloads: Boolean, selectBeforeOffline: Boolean = false, initiallyLoading: Boolean = false) {
        val paths = CopyOnWriteArrayList<String>()
        val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
        server.createContext("/") { exchange ->
            paths.add(exchange.requestURI.path)
            val body = if (exchange.requestURI.path.endsWith("/libraries"))
                """[{"id":"books","name":"Books","kind":"books","count":1}]""" else "[]"
            val bytes = body.toByteArray()
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        server.start()
        try {
            shadowOf(RuntimeEnvironment.getApplication().packageManager).setSystemFeature("test.feature", true)
            val vm = LibraryViewModel(RuntimeEnvironment.getApplication())
            LibraryViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }
                .set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
            val library = LibrarySummary("books", "Books", "books", 1)
            @Suppress("UNCHECKED_CAST")
            val state = LibraryViewModel::class.java.getDeclaredField("mutableState").apply { isAccessible = true }
                .get(vm) as MutableStateFlow<LibraryState>
            if (selectBeforeOffline) vm.setDownloadedOnly(true)
            state.value = LibraryState(allLibraries = listOf(library), libraries = listOf(library), selected = library,
                initialLoading = initiallyLoading, offlineOnly = true, downloadedOnly = true, grouping = PublicationGrouping.Books)
            if (userSelectedDownloads && !selectBeforeOffline) vm.setDownloadedOnly(true)
            ShadowSystemClock.advanceBy(Duration.ofSeconds(20))
            vm.refreshLibrarySummaries()
            val deadline = System.nanoTime() + 5_000_000_000L
            while (System.nanoTime() < deadline && paths.none { it.endsWith("/books") }) {
                shadowOf(Looper.getMainLooper()).idle()
                Thread.sleep(20)
            }
            shadowOf(Looper.getMainLooper()).idle()
            assertTrue("Recovery must reload the selected shelf, requests were $paths", paths.any { it.endsWith("/books") })
            assertFalse(vm.state.value.offlineOnly)
            org.junit.Assert.assertEquals("Recovery must restore the user's downloads-only choice", userSelectedDownloads, vm.state.value.downloadedOnly)
        } finally { server.stop(0) }
    }
}
