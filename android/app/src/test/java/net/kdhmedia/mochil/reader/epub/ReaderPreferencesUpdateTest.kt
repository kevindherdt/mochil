package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment

@RunWith(RobolectricTestRunner::class)
@org.robolectric.annotation.Config(sdk = [35])
class ReaderPreferencesUpdateTest {
    @Test fun readiumChromeSendsTransformsRatherThanStaleSnapshots() {
        val chrome = java.io.File("src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderChrome.kt").readText()
        val activity = java.io.File("src/main/java/net/kdhmedia/mochil/reader/epub/EpubReaderActivity.kt").readText()
        org.junit.Assert.assertTrue("Chrome events must carry transforms", chrome.contains("onPreferences: ((ReaderPreferences) -> ReaderPreferences) -> Unit"))
        org.junit.Assert.assertFalse("Activity must not save stale snapshots", activity.contains("preferencesStore.save("))
    }

    @Test fun concurrentStoresSerializeReadModifyWrite() {
        val context = RuntimeEnvironment.getApplication()
        context.getSharedPreferences("epub_reader_preferences", 0).edit().clear().commit()
        val first = EpubReaderPreferencesStore(context)
        val second = EpubReaderPreferencesStore(context)
        val entered = java.util.concurrent.CountDownLatch(1)
        val release = java.util.concurrent.CountDownLatch(1)
        val executor = java.util.concurrent.Executors.newFixedThreadPool(2)
        try {
            val a = executor.submit {
                first.update { current ->
                    entered.countDown()
                    check(release.await(5, java.util.concurrent.TimeUnit.SECONDS))
                    current.copy(font = ReaderFont.Serif)
                }
            }
            check(entered.await(5, java.util.concurrent.TimeUnit.SECONDS))
            val started = java.util.concurrent.CountDownLatch(1)
            val b = executor.submit {
                started.countDown()
                second.update { it.copy(tapZones = TapZones.Edges) }
            }
            check(started.await(5, java.util.concurrent.TimeUnit.SECONDS))
            release.countDown()
            a.get(5, java.util.concurrent.TimeUnit.SECONDS)
            b.get(5, java.util.concurrent.TimeUnit.SECONDS)
            assertEquals(ReaderFont.Serif, first.load().font)
            assertEquals(TapZones.Edges, first.load().tapZones)
        } finally {
            release.countDown()
            executor.shutdownNow()
        }
    }

    @Test fun independentSettingsUpdatesPreserveEachOther() {
        val context = RuntimeEnvironment.getApplication()
        context.getSharedPreferences("epub_reader_preferences", 0).edit().clear().commit()
        val first = EpubReaderPreferencesStore(context)
        val second = EpubReaderPreferencesStore(context)
        first.load(); second.load()
        val update = EpubReaderPreferencesStore::class.java.methods.firstOrNull { it.name == "update" && it.parameterCount == 1 }
        assertNotNull("Settings need an update operation based on current persisted values", update)
        update!!.invoke(first, { p: ReaderPreferences -> p.copy(font = ReaderFont.Serif) })
        update.invoke(second, { p: ReaderPreferences -> p.copy(tapZones = TapZones.Edges) })
        assertEquals(ReaderFont.Serif, first.load().font)
        assertEquals(TapZones.Edges, first.load().tapZones)
    }
}
