package net.kdhmedia.mochil.wiki

import java.io.ByteArrayInputStream
import java.io.File
import java.net.HttpURLConnection
import java.net.URL
import java.nio.file.Files
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.Job
import kotlinx.coroutines.CoroutineStart
import kotlinx.coroutines.launch
import org.junit.Assert.*
import org.junit.Test
import net.kdhmedia.mochil.auth.DeviceCredential

class WikiPdfDownloadsTest {
    @Test fun `failed replacement preserves the previous offline PDF at full quota`() = runBlocking {
        val root = Files.createTempDirectory("wiki-pdf-retain").toFile()
        try {
            var response = Response(pdf)
            var requests = 0
            val downloads = WikiPdfDownloads(root, maximumBytes = 1024, maximumFiles = 1,
                open = { _, _ -> requests++; response })
            val original = downloads.prepare(credential, "vault", entry)
            response = Response("Sign in".toByteArray(), "text/html")
            assertTrue(runCatching { downloads.prepare(credential, "vault", entry.copy(id = "new")) }.isFailure)
            assertArrayEquals(pdf, original.readBytes())
            assertEquals(original, downloads.prepare(credential, "vault", entry))
            assertEquals(2, requests) // Reopening the preserved revision needs no network.
            assertEquals(listOf(original), root.listFiles().orEmpty().toList())
            response = Response(pdf)
            val replacement = downloads.prepare(credential, "vault", entry.copy(id = "new"))
            assertFalse(original.exists())
            assertArrayEquals(pdf, replacement.readBytes())
            assertEquals(listOf(replacement), root.listFiles().orEmpty().toList())
        } finally { root.deleteRecursively() }
    }

    private val credential = DeviceCredential("https://fixture.invalid", "device", "secret")
    private val entry = WikiEntry("page", "pdf/id", "Notes/Manual.pdf", "Manual", "pdf", revision = "r1")
    private val pdf = "%PDF-1.7\nfixture".toByteArray()

    private open class Response(val bytes: ByteArray, val mime: String = "application/pdf", val announced: Long = bytes.size.toLong()) : HttpURLConnection(URL("https://fixture.invalid")) {
        var closed = false
        override fun connect() {}
        override fun disconnect() { closed = true }
        override fun usingProxy() = false
        override fun getResponseCode() = 200
        override fun getContentType() = mime
        override fun getContentLengthLong() = announced
        override fun getInputStream() = ByteArrayInputStream(bytes)
    }

    @Test fun `validated PDF caches by account library and revision with an encoded API path`() = runBlocking {
        val root = Files.createTempDirectory("wiki-pdf-test").toFile()
        try {
            val paths = mutableListOf<String>()
            val response = Response(pdf)
            val downloads = WikiPdfDownloads(root, maximumBytes = 1024, open = { _, path -> paths += path; response })
            val first = downloads.prepare(credential, "vault one", entry)
            assertArrayEquals(pdf, first.readBytes())
            assertEquals(first, downloads.prepare(credential, "vault one", entry))
            assertEquals(listOf("/api/android/v1/document-assets/vault%20one/pdf%2Fid/Manual.pdf"), paths)
            assertTrue(response.closed)
            assertNotEquals(first.name, downloads.prepare(credential, "vault one", entry.copy(revision = "r2")).name)
            assertNotEquals(wikiPdfIdentity(credential, "vault one", entry), wikiPdfIdentity(credential.copy(deviceId = "another"), "vault one", entry))
        } finally { root.deleteRecursively() }
    }

    @Test fun `wrong type oversized unknown length and truncated downloads never publish partial files`() = runBlocking {
        for (response in listOf(Response(pdf, "text/html"), Response(ByteArray(65), announced = -1), Response(pdf, announced = 40), Response("not a pdf".toByteArray()))) {
            val root = Files.createTempDirectory("wiki-pdf-failure").toFile()
            try {
                val downloads = WikiPdfDownloads(root, maximumBytes = 64, open = { _, _ -> response })
                assertTrue(runCatching { downloads.prepare(credential, "vault", entry) }.isFailure)
                assertTrue(root.listFiles().orEmpty().isEmpty())
                assertTrue(response.closed)
            } finally { root.deleteRecursively() }
        }
    }

    @Test fun `cache quota only prunes owned files and keeps other files untouched`() = runBlocking {
        val root = Files.createTempDirectory("wiki-pdf-quota").toFile()
        try {
            val unrelated = File(root, "keep.pdf").apply { writeText("User file") }
            val downloads = WikiPdfDownloads(root, maximumBytes = 1024, maximumFiles = 2, open = { _, _ -> Response(pdf) })
            repeat(4) { downloads.prepare(credential, "vault", entry.copy(id = "item$it")) }
            assertEquals(3, root.listFiles().orEmpty().size)
            assertEquals("User file", unrelated.readText())
        } finally { root.deleteRecursively() }
    }

    @Test fun `cancelled streaming disconnects and removes the partial without publishing`() = runBlocking {
        val root = Files.createTempDirectory("wiki-pdf-cancel").toFile()
        try {
            lateinit var job: Job
            val response = object : Response(pdf) {
                override fun getInputStream() = object : ByteArrayInputStream(pdf) {
                    override fun read(buffer: ByteArray, offset: Int, length: Int): Int {
                        job.cancel()
                        return super.read(buffer, offset, length)
                    }
                }
            }
            var published = false
            val downloads = WikiPdfDownloads(root, maximumBytes = 1024, open = { _, _ -> response })
            job = launch(start = CoroutineStart.LAZY) {
                downloads.prepare(credential, "vault", entry)
                published = true
            }
            job.start()
            job.join()
            assertTrue(job.isCancelled)
            assertFalse(published)
            assertTrue(response.closed)
            assertTrue(root.listFiles().orEmpty().isEmpty())
        } finally { root.deleteRecursively() }
    }

    @Test fun `crash partial cleanup is limited to component generated names`() = runBlocking {
        val root = Files.createTempDirectory("wiki-pdf-orphan").toFile()
        try {
            val orphan = File(root, "wiki-pdf-12345.part").apply { writeText("partial") }
            val unrelated = File(root, "other.part").apply { writeText("keep") }
            val downloads = WikiPdfDownloads(root, maximumBytes = 1024, open = { _, _ -> Response(pdf) })
            downloads.prepare(credential, "vault", entry)
            assertFalse(orphan.exists())
            assertEquals("keep", unrelated.readText())
        } finally { root.deleteRecursively() }
    }
}
