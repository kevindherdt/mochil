package net.kdhmedia.mochil.catalogue

import android.content.SharedPreferences
import java.io.File
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/**
 * Regression coverage for three defects a static reviewer flagged in
 * [PublicationDownloadStore]:
 *
 * 1. `@Synchronized` locks on the instance (`this`), but every production call site builds a
 *    fresh `PublicationDownloadStore(context)`. Two such instances share the on-disk index but
 *    never shared a monitor, so their critical sections never excluded each other at all -- not
 *    "sometimes flaky under contention", but deterministically unguarded. [`lost update across two instances`]
 *    proves this with explicit latches controlling the exact interleave, not timing.
 * 2. Malformed JSON in the preference made `decode()` return an empty list, and `record()`/
 *    `remove()` would then merge onto that empty list and commit it -- permanently destroying the
 *    corrupt (but potentially recoverable/diagnosable) raw data. [`corrupt index is preserved`]
 *    proves the fixed code refuses to do that.
 * 3. `preferences.edit().putString(...).commit()`'s boolean result was discarded, so a failed
 *    commit (e.g. disk full) was indistinguishable from success to every caller.
 *    [`failed commit is signalled`] proves the fixed code surfaces it.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class PublicationDownloadStoreIndexTest {
    private fun book(id: String) = BookSummary(
        id = id, libraryId = "lib", title = id, authors = emptyList(), series = "", seriesIndex = null,
        format = "epub", mediaType = "application/epub+zip", language = "", description = "", size = 4,
    )

    private fun file(name: String): File {
        val dir = File(RuntimeEnvironment.getApplication().filesDir, "publications").apply { mkdirs() }
        return File(dir, name).apply { writeBytes(byteArrayOf(1)) }
    }

    /**
     * Wraps a real [SharedPreferences] so a read can be paused mid-flight: `getString` signals
     * [started] the instant it is entered, then blocks on [resume] before delegating. This lets the
     * test force an exact interleave deterministically -- no sleeps, no timing-based flakiness --
     * rather than merely hammering threads and hoping a race shows up often enough.
     */
    private class PausingOnRead(
        private val delegate: SharedPreferences,
        private val started: CountDownLatch,
        private val resume: CountDownLatch,
    ) : SharedPreferences by delegate {
        override fun getString(key: String?, defValue: String?): String? {
            started.countDown()
            assertTrue("test setup: resume latch was never released", resume.await(5, TimeUnit.SECONDS))
            return delegate.getString(key, defValue)
        }
    }

    /** Wraps a real [SharedPreferences] so every commit reports failure without touching disk. */
    private class FailingCommit(private val delegate: SharedPreferences) : SharedPreferences by delegate {
        override fun edit(): SharedPreferences.Editor = FailingEditor(delegate.edit())
        private class FailingEditor(private val editorDelegate: SharedPreferences.Editor) : SharedPreferences.Editor by editorDelegate {
            // Delegated setters return the *real* editor, which would bypass this wrapper's commit().
            override fun putString(key: String?, value: String?): SharedPreferences.Editor {
                editorDelegate.putString(key, value)
                return this
            }

            override fun commit(): Boolean {
                editorDelegate.commit() // still exercises the real write path; only the reported result is false.
                return false
            }
        }
    }

    @Test fun `lost update across two instances -- record() on a paused instance does not erase a concurrent commit`() {
        val context = RuntimeEnvironment.getApplication()
        val backing = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        val decodeStarted = CountDownLatch(1)
        val resumeDecode = CountDownLatch(1)
        val storeA = PublicationDownloadStore(context, PausingOnRead(backing, decodeStarted, resumeDecode))
        val storeB = PublicationDownloadStore(context, backing)
        val workers = Executors.newFixedThreadPool(2)
        try {
            // Start A's record(): it enters decode(), calls getString, and now blocks on `resumeDecode`
            // having read the index as it stood before B ever ran.
            val resultA = workers.submit<Boolean> { storeA.record(book("book-a"), file("a.epub")) }
            assertTrue("A should have started reading the index", decodeStarted.await(5, TimeUnit.SECONDS))

            // B runs its *entire* record() to completion while A is parked mid-decode.
            //
            // Old code (@Synchronized on `this`): A and B are different instances, so B's
            // "synchronized" section was guarded by a different, uncontended monitor. B's full
            // decode-modify-commit cycle completes here. Then releasing A lets A resume with the
            // stale (pre-B) list it already read, add book-a, and commit [book-a] -- overwriting
            // book-b. The deterministic old-code result of this exact test is list() == [book-a]
            // only (size 1), book-b permanently lost, despite both record() calls reporting success.
            //
            // Fixed code: A and B share one lock keyed by the application context + preference name,
            // so B's record() blocks trying to acquire that same lock until A (which is already
            // holding it, parked inside decode()) finishes and releases it. B cannot even start its
            // own decode until A commits. The result below asserts that guarantee.
            val resultB = workers.submit<Boolean> { storeB.record(book("book-b"), file("b.epub")) }
            // Give B a real chance to run if (incorrectly) unblocked; on fixed code it stays blocked
            // on the shared lock the whole time, so this is not load-bearing for correctness, only
            // a sanity window before we deliberately unstick A.
            Thread.sleep(300)
            resumeDecode.countDown()

            assertTrue(resultA.get(5, TimeUnit.SECONDS))
            assertTrue(resultB.get(5, TimeUnit.SECONDS))
            val recorded = storeB.list().map { it.bookId }.toSet()
            assertEquals(setOf("book-a", "book-b"), recorded)
        } finally {
            workers.shutdownNow()
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `corrupt index is preserved -- record() throws instead of overwriting unparsable data`() {
        val context = RuntimeEnvironment.getApplication()
        val backing = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        backing.edit().putString("items", "{not valid json").commit()
        val store = PublicationDownloadStore(context, backing)
        try {
            try {
                store.record(book("new-book"), file("new.epub"))
                org.junit.Assert.fail("record() should refuse to merge onto a corrupt index")
            } catch (expected: PublicationIndexCorruptException) {
                // expected
            }
            // The corrupt raw value must still be exactly what it was -- nothing silently replaced
            // it with "[]" or with a one-item array containing only the new book.
            assertEquals("{not valid json", backing.getString("items", null))
        } finally {
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `corrupt index is preserved -- remove() does not commit over unparsable data`() {
        val context = RuntimeEnvironment.getApplication()
        val backing = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        backing.edit().putString("items", "not json either").commit()
        val store = PublicationDownloadStore(context, backing)
        val target = file("gone.epub")
        val item = DownloadedPublication("lib", "book", "Book", "", "epub", "application/epub+zip", target.absolutePath, 4)
        try {
            val removed = store.remove(item)
            // The file is still deleted (freeing space is independent of the index being readable),
            // but the operation must not report full success, and the corrupt bytes stay untouched.
            assertFalse(target.exists())
            assertFalse(removed)
            assertEquals("not json either", backing.getString("items", null))
        } finally {
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `failed commit is signalled -- record() reports false instead of claiming success`() {
        val context = RuntimeEnvironment.getApplication()
        val backing = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        val store = PublicationDownloadStore(context, FailingCommit(backing))
        try {
            assertFalse(store.record(book("will-not-persist"), file("x.epub")))
        } finally {
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `failed commit is signalled -- remove() reports false when the index commit fails`() {
        val context = RuntimeEnvironment.getApplication()
        val backing = context.getSharedPreferences("publication_downloads", android.content.Context.MODE_PRIVATE)
        val target = file("to-remove.epub")
        val bookSummary = book("to-remove")
        PublicationDownloadStore(context, backing).record(bookSummary, target)
        val store = PublicationDownloadStore(context, FailingCommit(backing))
        try {
            val item = requireNotNull(store.find(bookSummary.libraryId, bookSummary.id))
            assertFalse(store.remove(item))
            assertFalse(target.exists()) // file deletion still happened even though the index commit failed
        } finally {
            File(context.filesDir, "publications").deleteRecursively()
        }
    }

    @Test fun `happy path -- record() and remove() report true on success`() {
        val context = RuntimeEnvironment.getApplication()
        val store = PublicationDownloadStore(context)
        val target = file("ok.epub")
        val bookSummary = book("ok")
        try {
            assertTrue(store.record(bookSummary, target))
            val recorded = requireNotNull(store.find(bookSummary.libraryId, bookSummary.id))
            assertTrue(store.remove(recorded))
            assertNull(store.find(bookSummary.libraryId, bookSummary.id))
        } finally {
            File(context.filesDir, "publications").deleteRecursively()
        }
    }
}
