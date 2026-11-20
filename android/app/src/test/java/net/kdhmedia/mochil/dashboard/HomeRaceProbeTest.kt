package net.kdhmedia.mochil.dashboard

import android.os.Looper
import com.sun.net.httpserver.HttpServer
import java.net.InetSocketAddress
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.Executors
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.catalogue.BookSummary
import net.kdhmedia.mochil.auth.DeviceCredentialStore
import android.content.Context
import org.robolectric.annotation.Implements
import org.robolectric.annotation.Implementation
import org.junit.Assert.*
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

@Implements(DeviceCredentialStore::class)
class DashboardCredentialStoreShadow {
 @Implementation fun __constructor__(context: Context) {}
 @Implementation fun load(): DeviceCredential? = null
}

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35], shadows = [DashboardCredentialStoreShadow::class])
class HomeRaceProbeTest {
 @Test fun failedRefreshKeepsPreviousContentUntilRecovery() {
  val fail = java.util.concurrent.atomic.AtomicBoolean(false)
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.createContext("/") { ex ->
   if (fail.get()) { ex.sendResponseHeaders(503, -1); ex.close() }
   else {
    val body = if (ex.requestURI.path.endsWith("/home")) """{"reading":[{"id":"book","library_id":"books","title":"Book","authors":[]}],"listening":[],"watching":[]}""" else "[]"
    val bytes = body.toByteArray(); ex.sendResponseHeaders(200, bytes.size.toLong()); ex.responseBody.use { it.write(bytes) }
   }
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   fun waitFor(condition: () -> Boolean) {
    val deadline = System.nanoTime() + 5_000_000_000L
    while (!condition() && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
    shadowOf(Looper.getMainLooper()).idle()
    assertTrue("Timed out waiting for dashboard", condition())
   }
   vm.refresh(); waitFor { !vm.state.value.loading && vm.state.value.reading.isNotEmpty() }
   fail.set(true); vm.refresh(); waitFor { !vm.state.value.loading && vm.state.value.error != null }
   assertEquals("Book", vm.state.value.reading.single().book.title)
   fail.set(false); vm.refresh(); waitFor { !vm.state.value.loading && vm.state.value.error == null }
   assertEquals("Book", vm.state.value.reading.single().book.title)
  } finally { server.stop(0) }
 }

 @Test fun temporary404DoesNotPermanentlyDisableCombinedHome() {
  var calls = 0
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.createContext("/") { ex ->
   calls++
   val code = if (calls == 1) 404 else 200
   val body = """{"reading":[],"listening":[],"watching":[]} """.toByteArray()
   ex.sendResponseHeaders(code, body.size.toLong()); ex.responseBody.use { it.write(body) }
  }; server.start()
  try {
   val client = DashboardClient()
   val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test")
   assertNull(client.home(credential))
   assertNotNull("Home should retry the endpoint after a temporary 404", client.home(credential))
  } finally { server.stop(0) }
 }
 @Test fun queuedRefreshPublishesFirstSnapshotWhileSecondRequestIsBlocked() {
  val firstEntered = CountDownLatch(1); val releaseFirst = CountDownLatch(1)
  val secondEntered = CountDownLatch(1); val releaseSecond = CountDownLatch(1)
  val calls = java.util.concurrent.atomic.AtomicInteger()
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.executor = Executors.newCachedThreadPool { r -> Thread(r).apply { isDaemon = true } }
  server.createContext("/") { ex ->
   if (ex.requestURI.path.endsWith("/home")) {
    if (calls.incrementAndGet() == 1) { firstEntered.countDown(); releaseFirst.await(10, TimeUnit.SECONDS) }
    else { secondEntered.countDown(); releaseSecond.await(10, TimeUnit.SECONDS) }
   }
   val body = if (ex.requestURI.path.endsWith("/home")) """{"reading":[{"id":"book","library_id":"books","title":"Book","authors":[]}],"listening":[],"watching":[]}""" else "[]"
   val bytes = body.toByteArray(); ex.sendResponseHeaders(200, bytes.size.toLong()); ex.responseBody.use { it.write(bytes) }
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   vm.refresh(); shadowOf(Looper.getMainLooper()).idle()
   assertTrue(firstEntered.await(3, TimeUnit.SECONDS))
   vm.refresh(); releaseFirst.countDown()
   val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
   while (secondEntered.count > 0 && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertEquals("queued request should start", 0L, secondEntered.count)
   shadowOf(Looper.getMainLooper()).idle()
   assertEquals("first snapshot should be visible while the follow-up is blocked", "Book", vm.state.value.reading.singleOrNull()?.book?.title)
   assertFalse("first snapshot should end initial loading", vm.state.value.loading)
  } finally { releaseFirst.countDown(); releaseSecond.countDown(); server.stop(0) }
 }

 @Test fun queuedRefreshDoesNotStarveHeldAndFinishedWhileFollowUpIsBlocked() {
  val firstEntered = CountDownLatch(1); val releaseFirst = CountDownLatch(1)
  val secondEntered = CountDownLatch(1); val releaseSecond = CountDownLatch(1)
  val calls = java.util.concurrent.atomic.AtomicInteger()
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.executor = Executors.newCachedThreadPool { r -> Thread(r).apply { isDaemon = true } }
  server.createContext("/") { ex ->
   val path = ex.requestURI.path; val query = ex.requestURI.query.orEmpty()
   if (path.endsWith("/home")) {
    if (calls.incrementAndGet() == 1) { firstEntered.countDown(); releaseFirst.await(10, TimeUnit.SECONDS) }
    else { secondEntered.countDown(); releaseSecond.await(10, TimeUnit.SECONDS) }
   }
   val body = when {
    path.endsWith("/home") -> """{"reading":[],"listening":[],"watching":[]}"""
    query.contains("held=true") -> """[{"id":"held","library_id":"books","title":"Held","authors":[]}]"""
    query.contains("finished=true") -> """[{"id":"finished","library_id":"books","title":"Finished","authors":[]}]"""
    else -> "[]"
   }
   val bytes = body.toByteArray(); ex.sendResponseHeaders(200, bytes.size.toLong()); ex.responseBody.use { it.write(bytes) }
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   vm.refresh(); shadowOf(Looper.getMainLooper()).idle(); assertTrue(firstEntered.await(3, TimeUnit.SECONDS))
   vm.refresh(); releaseFirst.countDown()
   val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
   while ((secondEntered.count > 0 || vm.state.value.finished.isEmpty()) && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertEquals("queued request should start", 0L, secondEntered.count)
   assertEquals("Held", vm.state.value.held.singleOrNull()?.book?.title)
   assertEquals("Finished", vm.state.value.finished.singleOrNull()?.book?.title)
  } finally { releaseFirst.countDown(); releaseSecond.countDown(); server.stop(0) }
 }

 @Test fun queuedRefreshPublishesFallbackWhileFollowUpIsBlocked() {
  val fallbackEntered = CountDownLatch(1); val releaseFallback = CountDownLatch(1)
  val secondEntered = CountDownLatch(1); val releaseSecond = CountDownLatch(1)
  val calls = java.util.concurrent.atomic.AtomicInteger()
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.executor = Executors.newCachedThreadPool { r -> Thread(r).apply { isDaemon = true } }
  server.createContext("/") { ex ->
   val path = ex.requestURI.path
   if (path.endsWith("/home")) {
    if (calls.incrementAndGet() == 1) { ex.sendResponseHeaders(404, -1); ex.close(); return@createContext }
    secondEntered.countDown(); releaseSecond.await(10, TimeUnit.SECONDS)
   } else if (path.endsWith("/reading") && ex.requestURI.query?.contains("held=") != true && ex.requestURI.query?.contains("finished=") != true) {
    fallbackEntered.countDown(); releaseFallback.await(10, TimeUnit.SECONDS)
   }
   val body = if (path.endsWith("/home")) """{"reading":[],"listening":[],"watching":[]}""" else if (path.endsWith("/reading") && ex.requestURI.query?.contains("held=") != true && ex.requestURI.query?.contains("finished=") != true) """[{"id":"book","library_id":"books","title":"Fallback","authors":[]}]""" else "[]"
   val bytes = body.toByteArray(); ex.sendResponseHeaders(200, bytes.size.toLong()); ex.responseBody.use { it.write(bytes) }
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   vm.refresh(); shadowOf(Looper.getMainLooper()).idle()
   val fallbackDeadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
   while (fallbackEntered.count > 0 && System.nanoTime() < fallbackDeadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertEquals("fallback request should start", 0L, fallbackEntered.count)
   vm.refresh(); releaseFallback.countDown()
   val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
   while (secondEntered.count > 0 && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertEquals("queued request should start", 0L, secondEntered.count)
   shadowOf(Looper.getMainLooper()).idle()
   assertEquals("Fallback", vm.state.value.reading.singleOrNull()?.book?.title)
   assertFalse("fallback should end initial loading", vm.state.value.loading)
  } finally { releaseFallback.countDown(); releaseSecond.countDown(); server.stop(0) }
 }

 @Test fun delayedHeldAndFinishedStartedDuringResumeCannotRestoreResumedBook() {
  val statusEntered = CountDownLatch(1); val releaseStatus = CountDownLatch(1)
  val finishedEntered = CountDownLatch(1); val releaseFinished = CountDownLatch(1); val finishedSent = CountDownLatch(1)
  val followupEntered = CountDownLatch(1); val releaseFollowup = CountDownLatch(1)
  val homeCalls = java.util.concurrent.atomic.AtomicInteger()
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.executor = Executors.newCachedThreadPool { r -> Thread(r).apply { isDaemon = true } }
  server.createContext("/") { ex ->
   val path = ex.requestURI.path; val query = ex.requestURI.query.orEmpty()
   if (ex.requestMethod != "GET") { statusEntered.countDown(); releaseStatus.await(10, TimeUnit.SECONDS) }
   if (path.endsWith("/home") && homeCalls.incrementAndGet() == 2) { followupEntered.countDown(); releaseFollowup.await(10, TimeUnit.SECONDS) }
   if (query.contains("finished=true")) { finishedEntered.countDown(); releaseFinished.await(10, TimeUnit.SECONDS) }
   val body = when {
    path.endsWith("/home") -> """{"reading":[],"listening":[],"watching":[]}"""
    query.contains("held=true") || query.contains("finished=true") -> """[{"id":"id","library_id":"lib","title":"Book","authors":[]}]"""
    else -> "[]"
   }.toByteArray()
   ex.sendResponseHeaders(200, body.size.toLong()); ex.responseBody.use { it.write(body) }
   if (query.contains("finished=true")) finishedSent.countDown()
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   fun waitFor(latch: CountDownLatch) {
    val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(5)
    while (latch.count > 0 && System.nanoTime() < deadline) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
    assertEquals("Timed out waiting for server", 0L, latch.count)
   }
   vm.resumeHeld(BookSummary("id", "lib", "Book", emptyList(), "", null, "", "", "", "", 0))
   waitFor(statusEntered)
   vm.refresh(); waitFor(finishedEntered)
   releaseStatus.countDown(); waitFor(followupEntered)
   releaseFinished.countDown(); waitFor(finishedSent)
   repeat(100) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertTrue("stale held response restored resumed book", vm.state.value.held.isEmpty())
   assertTrue("stale finished response restored resumed book", vm.state.value.finished.isEmpty())
  } finally { releaseStatus.countDown(); releaseFinished.countDown(); releaseFollowup.countDown(); server.stop(0) }
 }

 @Test fun resumeDuringRefreshMustNotLeaveHomeLoading() {
  val entered = CountDownLatch(1); val release = CountDownLatch(1); val status = CountDownLatch(1)
  val homeCalls = java.util.concurrent.atomic.AtomicInteger()
  val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0)
  server.executor = Executors.newCachedThreadPool { r -> Thread(r).apply { isDaemon = true } }
  server.createContext("/") { ex ->
   if (ex.requestURI.path.endsWith("/home")) { homeCalls.incrementAndGet(); entered.countDown(); release.await(5, TimeUnit.SECONDS) }
   else if (ex.requestMethod != "GET") status.countDown()
   val body = if (ex.requestURI.path.endsWith("/home")) """{"reading":[],"listening":[],"watching":[]} """ else "[]"
   val bytes=body.toByteArray(); ex.sendResponseHeaders(200, bytes.size.toLong()); ex.responseBody.use { it.write(bytes) }
  }; server.start()
  try {
   val vm = DashboardViewModel(RuntimeEnvironment.getApplication())
   DashboardViewModel::class.java.getDeclaredField("credential").apply { isAccessible = true }.set(vm, DeviceCredential("http://127.0.0.1:${server.address.port}", "test", "test"))
   vm.refresh(); shadowOf(Looper.getMainLooper()).idle()
   assertTrue(entered.await(3, TimeUnit.SECONDS))
   vm.resumeHeld(BookSummary("id", "lib", "book", emptyList(), "", null, "", "", "", "", 0))
   shadowOf(Looper.getMainLooper()).idle(); assertTrue(status.await(3, TimeUnit.SECONDS))
   repeat(50) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   release.countDown()
   repeat(100) { shadowOf(Looper.getMainLooper()).idle(); Thread.sleep(10) }
   assertFalse("Home remains loading after resume overlaps a refresh", vm.state.value.loading)
   assertEquals("Resume must trigger a fresh request", 2, homeCalls.get())
  } finally { release.countDown(); server.stop(0) }
 }
}
