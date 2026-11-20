package net.kdhmedia.mochil.catalogue

import com.sun.net.httpserver.HttpServer
import java.io.File
import java.net.InetSocketAddress
import java.util.concurrent.atomic.AtomicInteger
import kotlinx.coroutines.runBlocking
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.opds.opdsBookId
import net.kdhmedia.mochil.opds.opdsTarget
import net.kdhmedia.mochil.opds.recordOpdsDownload
import net.kdhmedia.mochil.source.SourceItem
import net.kdhmedia.mochil.reader.comic.ComicReaderActivity
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import android.content.ContentProvider
import android.content.ContentValues
import android.database.Cursor
import android.net.Uri
import org.robolectric.Robolectric
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/** Offline-cache identity: a downloaded file must belong to exactly one (library, book) key. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PublicationCacheIdentityTest {
    private val context = RuntimeEnvironment.getApplication()
    private val requests = AtomicInteger()
    private val server = HttpServer.create(InetSocketAddress("127.0.0.1", 0), 0).apply {
        // The body is the request path, so every (library, book) URL yields distinct, checkable bytes.
        createContext("/") { exchange ->
            requests.incrementAndGet()
            val bytes = exchange.requestURI.rawPath.toByteArray()
            exchange.sendResponseHeaders(200, bytes.size.toLong())
            exchange.responseBody.use { it.write(bytes) }
        }
        start()
    }
    private val credential = DeviceCredential("http://127.0.0.1:${server.address.port}", "device", "token")
    private val publications get() = File(context.filesDir, "publications")

    @After fun cleanup() {
        server.stop(0)
        publications.deleteRecursively()
        File(context.filesDir, "opds").deleteRecursively()
        context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE).edit().clear().commit()
    }

    private fun book(id: String, library: String = "lib-a", title: String = "Same Title") = BookSummary(
        id = id, libraryId = library, title = title, authors = emptyList(), series = "", seriesIndex = null,
        format = "cbz", mediaType = "application/vnd.comicbook+zip", language = "", description = "", size = 4,
    )

    private fun open(book: BookSummary): File {
        val intent = runBlocking { PublicationOpener(context).prepare(credential, book) }
        return File(requireNotNull(intent.getStringExtra(ComicReaderActivity.EXTRA_PATH)))
    }

    private fun legacyName(book: BookSummary) =
        "${book.title}-${book.id.hashCode().toUInt().toString(16)}.${book.format}"

    private fun legacyFile(book: BookSummary, content: ByteArray): File =
        File(publications.apply { mkdirs() }, legacyName(book)).apply { writeBytes(content) }

    @Test fun `same title format and id in two libraries get different files and bytes`() {
        val first = open(book("42", "lib-a"))
        val second = open(book("42", "lib-b"))
        assertNotEquals(first.absolutePath, second.absolutePath)
        assertEquals(2, requests.get())
        assertTrue(first.readText().contains("lib-a"))
        assertTrue(second.readText().contains("lib-b"))
    }

    @Test fun `ids with equal Java hash codes do not share a file`() {
        assertEquals("Aa".hashCode(), "BB".hashCode())
        val first = open(book("Aa"))
        val second = open(book("BB"))
        assertNotEquals(first.absolutePath, second.absolutePath)
        assertEquals(2, requests.get())
        assertTrue(first.readText().endsWith("/Aa"))
        assertTrue(second.readText().endsWith("/BB"))
    }

    @Test fun `a legacy indexed entry keeps its path and is not downloaded again`() {
        val legacy = book("legacy-1")
        // Deliberately not at the name the old formula would produce: only the index can find it.
        val file = File(publications.apply { mkdirs() }, "renamed-by-an-older-build.cbz").apply { writeBytes(byteArrayOf(9, 9, 9, 9)) }
        assertTrue(PublicationDownloadStore(context).record(legacy, file))
        val opened = open(legacy)
        assertEquals(file.absolutePath, opened.absolutePath)
        assertEquals(0, requests.get())
        assertArrayEquals(byteArrayOf(9, 9, 9, 9), file.readBytes())
        assertEquals(file.absolutePath, PublicationDownloadStore(context).find(legacy.libraryId, legacy.id)?.path)
    }

    @Test fun `an unindexed file at a legacy name for another key is never reused`() {
        val owner = book("77", "lib-a")
        val stranger = book("77", "lib-b")
        val ownerFile = legacyFile(owner, byteArrayOf(1, 1, 1, 1))
        PublicationDownloadStore(context).record(owner, ownerFile)
        val opened = open(stranger)
        assertNotEquals(ownerFile.absolutePath, opened.absolutePath)
        assertEquals(1, requests.get())
        assertTrue(opened.readText().contains("lib-b"))
        assertArrayEquals(byteArrayOf(1, 1, 1, 1), ownerFile.readBytes())
    }

    @Test fun `an unindexed leftover at the legacy name is not reused either`() {
        val wanted = book("55")
        val leftover = legacyFile(wanted, byteArrayOf(7, 7, 7, 7))
        val opened = open(wanted)
        assertNotEquals(leftover.absolutePath, opened.absolutePath)
        assertEquals(1, requests.get())
        assertTrue(opened.readText().endsWith("/55"))
    }

    @Test fun `removing one of two legacy entries sharing a path keeps the file for the other`() {
        val a = book("shared", "lib-a")
        val b = book("shared", "lib-b")
        val shared = legacyFile(a, byteArrayOf(5, 5))
        val store = PublicationDownloadStore(context)
        store.record(a, shared)
        store.record(b, shared)
        assertTrue(store.remove(requireNotNull(store.find("lib-a", "shared"))))
        assertTrue("file still referenced by lib-b", shared.isFile)
        assertEquals(shared.absolutePath, store.find("lib-b", "shared")?.path)
        assertTrue(store.remove(requireNotNull(store.find("lib-b", "shared"))))
        assertFalse(shared.exists())
    }

    @Test fun `OPDS blank-id items from different URLs do not collide`() {
        val store = PublicationDownloadStore(context)
        val directory = File(context.filesDir, "opds")
        fun target(url: String) = PublicationFileNaming.resolveTarget(
            store, directory, PublicationFileNaming.Source.OPDS, "opds:conn-1",
            bookId = url.hashCode().toUInt().toString(16), title = "Untitled", extension = "epub", identity = url,
        )
        assertNotEquals(target("https://x.example/a.epub"), target("https://x.example/b.epub"))
    }

    @Test fun `OPDS same id and title on two connections get different files`() {
        val store = PublicationDownloadStore(context)
        val directory = File(context.filesDir, "opds")
        fun target(connection: String) = PublicationFileNaming.resolveTarget(
            store, directory, PublicationFileNaming.Source.OPDS, "opds:$connection", bookId = "urn:1", title = "T", extension = "epub",
        )
        assertNotEquals(target("one"), target("two"))
        assertNotEquals(
            PublicationFileNaming.resolveTarget(store, directory, PublicationFileNaming.Source.OPDS, "x", "1", "T", "epub"),
            PublicationFileNaming.resolveTarget(store, directory, PublicationFileNaming.Source.CATALOGUE, "x", "1", "T", "epub"),
        )
    }

    @Test fun `an indexed OPDS legacy path is kept`() {
        val store = PublicationDownloadStore(context)
        val directory = File(context.filesDir, "opds").apply { mkdirs() }
        val legacy = File(directory, "1a2b3c.epub").apply { writeBytes(byteArrayOf(1)) }
        store.record(book("urn:9", "opds:conn-1").copy(format = "epub"), legacy)
        val target = PublicationFileNaming.resolveTarget(store, directory, PublicationFileNaming.Source.OPDS, "opds:conn-1", "urn:9", "T", "epub")
        assertEquals(legacy.absolutePath, target.absolutePath)
    }

    @Test fun `names use only safe characters and stay short`() {
        val store = PublicationDownloadStore(context)
        val nasty = "../../etc/pass\u0000wd \u00e9\u4e2d\n" + "x".repeat(500)
        for (source in PublicationFileNaming.Source.values()) {
            val name = PublicationFileNaming.resolveTarget(store, File(context.filesDir, "p"), source, "lib/../x", "id/../" + "y".repeat(500), nasty, "epub").name
            assertTrue(name, name.matches(Regex("[A-Za-z0-9._ -]+")))
            assertTrue(name, name.length <= 128)
            assertTrue(name, name.endsWith(".epub"))
        }
    }

    // ---- OPDS blank-id items whose acquisition URLs have equal String.hashCode ----

    private fun blankIdItem(url: String) = SourceItem(
        id = "", libraryId = "opds:conn-1", title = "Untitled", artworkUrl = null,
        mediaType = "application/epub+zip", acquisitionUrl = url,
    )

    /** What OpdsViewModel.download does after the HTTP fetch: resolve the target, write the bytes, record. */
    private fun downloadOpds(item: SourceItem, bytes: ByteArray): File {
        val directory = File(context.filesDir, "opds").apply { mkdirs() }
        val target = opdsTarget(PublicationDownloadStore(context), "conn-1", item, directory, "epub")
        if (!(target.isFile && target.length() > 0)) target.writeBytes(bytes)
        return recordOpdsDownload(context, "conn-1", item, target, "epub")
    }

    @Test fun `OPDS blank-id items with colliding URL hashes keep their own files and index entries`() {
        val urlA = "https://x.example/get/Aa"
        val urlB = "https://x.example/get/BB"
        assertEquals(urlA.hashCode(), urlB.hashCode())
        val a = blankIdItem(urlA)
        val b = blankIdItem(urlB)
        assertNotEquals(opdsBookId(a), opdsBookId(b))
        assertTrue(opdsBookId(a).matches(Regex("u:[0-9a-f]{64}")))

        val fileA = downloadOpds(a, "bytes-of-A".toByteArray())
        val fileB = downloadOpds(b, "bytes-of-B".toByteArray())

        assertNotEquals(fileA.absolutePath, fileB.absolutePath)
        assertEquals("bytes-of-A", fileA.readText())
        assertEquals("bytes-of-B", fileB.readText())
        val store = PublicationDownloadStore(context)
        assertEquals(fileA.absolutePath, store.find("opds:conn-1", opdsBookId(a))?.path)
        assertEquals(fileB.absolutePath, store.find("opds:conn-1", opdsBookId(b))?.path)
        assertEquals(2, store.list().size)
    }

    @Test fun `OPDS blank-id legacy hash-keyed entry is not matched and its file is left alone`() {
        val url = "https://x.example/get/Aa"
        val item = blankIdItem(url)
        val directory = File(context.filesDir, "opds").apply { mkdirs() }
        val legacyFile = File(directory, "legacy.epub").apply { writeBytes(byteArrayOf(1, 2, 3)) }
        val legacyKey = url.hashCode().toUInt().toString(16)
        PublicationDownloadStore(context).record(book(legacyKey, "opds:conn-1").copy(format = "epub"), legacyFile)

        val fresh = downloadOpds(item, "new".toByteArray())

        assertNotEquals(legacyFile.absolutePath, fresh.absolutePath)
        assertEquals("new", fresh.readText())
        assertArrayEquals(byteArrayOf(1, 2, 3), legacyFile.readBytes())
        assertEquals(legacyFile.absolutePath, PublicationDownloadStore(context).find("opds:conn-1", legacyKey)?.path)
    }

    // ---- remove() must act on the current index entry, not the caller's snapshot ----

    @Test fun `remove with a stale snapshot deletes the currently indexed file and entry`() {
        val store = PublicationDownloadStore(context)
        val b = book("stale-1")
        val x = legacyFile(b, byteArrayOf(1)) // legacy path the snapshot still points at
        val y = File(publications, "repointed.cbz").apply { writeBytes(byteArrayOf(2)) }
        store.record(b, x)
        val stale = requireNotNull(store.find(b.libraryId, b.id))
        store.record(b, y) // index repointed to Y after the snapshot was taken

        assertTrue(store.remove(stale))

        assertFalse("current file Y must not be stranded", y.exists())
        assertNull(store.find(b.libraryId, b.id))
        assertTrue("stale path X is no longer this entry's file and is left alone", x.exists())
    }

    @Test fun `remove of a key that is no longer indexed deletes nothing`() {
        val store = PublicationDownloadStore(context)
        val b = book("gone-1")
        val x = legacyFile(b, byteArrayOf(1))
        store.record(b, x)
        val stale = requireNotNull(store.find(b.libraryId, b.id))
        assertTrue(store.remove(stale))
        assertFalse(x.exists())
        // A concurrent download put a file at the same path but has not recorded it yet.
        x.writeBytes(byteArrayOf(9))
        assertTrue(store.remove(stale))
        assertTrue("an unindexed key must never delete the caller's stale path", x.exists())
    }

    @Test fun `remove with a corrupt index still reports failure and keeps the raw value`() {
        val prefs = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        val store = PublicationDownloadStore(context)
        val b = book("corrupt-1")
        val x = legacyFile(b, byteArrayOf(1))
        store.record(b, x)
        val snapshot = requireNotNull(store.find(b.libraryId, b.id))
        prefs.edit().putString("items", "{not json").commit()
        assertFalse(store.remove(snapshot))
        assertEquals("{not json", prefs.getString("items", null))
    }

    // ---- mirror (externalUri) handling on remove ----

    class RecordingProvider : ContentProvider() {
        override fun onCreate() = true
        override fun query(uri: Uri, p: Array<String>?, s: String?, a: Array<String>?, o: String?): Cursor? = null
        override fun getType(uri: Uri): String? = null
        override fun insert(uri: Uri, values: ContentValues?): Uri? = null
        override fun update(uri: Uri, v: ContentValues?, s: String?, a: Array<String>?) = 0
        override fun delete(uri: Uri, s: String?, a: Array<String>?): Int { deleted += uri.toString(); return 1 }
        companion object { val deleted = mutableListOf<String>() }
    }

    private fun mirrorSetup(): Pair<PublicationDownloadStore, File> {
        RecordingProvider.deleted.clear()
        Robolectric.buildContentProvider(RecordingProvider::class.java).create("mirror.test")
        return PublicationDownloadStore(context) to legacyFile(book("shared-file"), byteArrayOf(4, 4))
    }

    @Test fun `removing an entry with a mirror while a sibling shares the file deletes the mirror but keeps the file`() {
        val (store, shared) = mirrorSetup()
        val mirrorA = "content://mirror.test/doc/a"
        store.record(book("a"), shared, mirrorA)
        store.record(book("b"), shared) // sibling: same private file, no mirror of its own

        assertTrue(store.remove(requireNotNull(store.find("lib-a", "a"))))

        // Documented behaviour: the mirror is the removed entry's own copy, so it is deleted even though
        // the private file stays because "b" still references it.
        assertEquals(listOf(mirrorA), RecordingProvider.deleted)
        assertTrue(shared.isFile)
        assertNull(store.find("lib-a", "a"))
        assertEquals(shared.absolutePath, store.find("lib-a", "b")?.path)
    }

    @Test fun `a mirror still referenced by a sibling entry is kept`() {
        val (store, shared) = mirrorSetup()
        val mirror = "content://mirror.test/doc/shared"
        store.record(book("a"), shared, mirror)
        store.record(book("b"), shared, mirror)

        assertTrue(store.remove(requireNotNull(store.find("lib-a", "a"))))

        assertTrue(RecordingProvider.deleted.isEmpty())
        assertTrue(shared.isFile)
        assertEquals(mirror, store.find("lib-a", "b")?.externalUri)
    }

    @Test fun `remove with a stale snapshot deletes the current entry's mirror, not the snapshot's`() {
        val (store, _) = mirrorSetup()
        val b = book("m")
        val x = File(publications, "x.cbz").apply { writeBytes(byteArrayOf(1)) }
        val y = File(publications, "y.cbz").apply { writeBytes(byteArrayOf(2)) }
        store.record(b, x, "content://mirror.test/doc/old")
        val stale = requireNotNull(store.find(b.libraryId, b.id))
        store.record(b, y, "content://mirror.test/doc/new")

        assertTrue(store.remove(stale))

        assertEquals(listOf("content://mirror.test/doc/new"), RecordingProvider.deleted)
        assertFalse(y.exists())
    }
}
