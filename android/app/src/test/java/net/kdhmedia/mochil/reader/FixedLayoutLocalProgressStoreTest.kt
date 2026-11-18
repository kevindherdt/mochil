package net.kdhmedia.mochil.reader

import android.content.Context
import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class FixedLayoutLocalProgressStoreTest {
    @Test fun `PDF and comic positions remain isolated in their existing stores`() {
        assertEquals("pdf_reader", FixedLayoutReaderKind.Pdf.preferencesName)
        assertEquals("comic_reader", FixedLayoutReaderKind.Comic.preferencesName)
        assertNotEquals(FixedLayoutReaderKind.Pdf.preferencesName, FixedLayoutReaderKind.Comic.preferencesName)
    }

    @Test fun `fixed layout page keys remain backward compatible`() {
        assertEquals("page:library:book", fixedLayoutPageKey("library:book"))
    }

    @Test fun `saveNow persists a page durably and load reads it back`() {
        val context = RuntimeEnvironment.getApplication()
        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)

        assertTrue(store.saveNow("library:book", 42))

        assertEquals(42, store.load("library:book", maximumPage = 100))
    }

    @Test fun `save and saveNow keep PDF and comic progress isolated`() {
        val context = RuntimeEnvironment.getApplication()
        val pdfStore = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)
        val comicStore = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Comic)

        pdfStore.saveNow("library:book", 10)
        comicStore.saveNow("library:book", 99)

        assertEquals(10, pdfStore.load("library:book", maximumPage = 200))
        assertEquals(99, comicStore.load("library:book", maximumPage = 200))
    }

    @Test fun `a write never leaves a partial temp file behind on success`() {
        val context = RuntimeEnvironment.getApplication()
        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)

        store.saveNow("library:book", 7)

        val stateFile = File(context.noBackupFilesDir, FixedLayoutReaderKind.Pdf.fileName)
        val temporary = File(context.noBackupFilesDir, ".${stateFile.name}.part")
        assertTrue("the real state file should exist after a successful save", stateFile.isFile)
        assertFalse("no stray .part temp file should remain after a successful save", temporary.isFile)
    }

    @Test fun `a truncated or corrupt state file degrades to page 0 instead of crashing`() {
        val context = RuntimeEnvironment.getApplication()
        val stateFile = File(context.noBackupFilesDir, FixedLayoutReaderKind.Pdf.fileName)
        stateFile.parentFile?.mkdirs()
        stateFile.writeText("{not valid json")

        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)

        assertEquals(0, store.load("library:book", maximumPage = 50))
    }

    @Test fun `load clamps a corrupt or out-of-range page into the valid window`() {
        val context = RuntimeEnvironment.getApplication()
        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)

        store.saveNow("library:book", 999)

        assertEquals(50, store.load("library:book", maximumPage = 50))
    }

    @Test fun `migrateLegacyKey lifts a page and clears the legacy flags exactly once`() {
        val context = RuntimeEnvironment.getApplication()
        val legacyPreferences = context.getSharedPreferences(FixedLayoutReaderKind.Pdf.preferencesName, Context.MODE_PRIVATE)
        legacyPreferences.edit()
            .putInt("page:old-key", 17)
            .putBoolean("finished:old-key", true)
            .commit()

        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)
        store.migrateLegacyKey("old-key", "new-key")

        assertEquals(17, store.load("new-key", maximumPage = 100))
        assertFalse(legacyPreferences.contains("page:old-key"))
        assertFalse(legacyPreferences.contains("finished:old-key"))

        // A second migration call is a no-op: the new key already has a value.
        legacyPreferences.edit().putInt("page:old-key", 5).commit()
        store.migrateLegacyKey("old-key", "new-key")
        assertEquals(17, store.load("new-key", maximumPage = 100))
    }

    @Test fun `migrateLegacyKey does nothing when there is no legacy data`() {
        val context = RuntimeEnvironment.getApplication()
        val store = FixedLayoutLocalProgressStore(context, FixedLayoutReaderKind.Pdf)

        store.migrateLegacyKey("missing-key", "new-key")

        assertEquals(0, store.load("new-key", maximumPage = 100))
    }
}
