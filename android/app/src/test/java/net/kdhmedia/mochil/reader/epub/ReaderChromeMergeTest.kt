package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config

/** Reading progress is one None/Bar/Time choice and Sharper text is Off/On; both read the older saved shapes. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [35])
class ReaderChromeMergeTest {
    private val context get() = RuntimeEnvironment.getApplication()
    private fun prefs() = context.getSharedPreferences("epub_reader_preferences", 0).edit().clear()
    private fun load(eInk: Boolean = false) = EpubReaderPreferencesStore(context, eInk = eInk).load()

    @Test fun aNewInstallShowsNoProgress() {
        prefs().commit()
        assertEquals(ReadingProgress.None, load().progress)
    }

    @Test fun oldProgressBarOnlyBecomesBar() {
        prefs().putBoolean("show_progress_bar", true).commit()
        assertEquals(ReadingProgress.Bar, load().progress)
    }

    @Test fun oldChapterOrBookTimeBecomesTime() {
        prefs().putBoolean("show_chapter_time_remaining", true).commit()
        assertEquals(ReadingProgress.Time, load().progress)
        prefs().putBoolean("show_book_time_remaining", true).commit()
        assertEquals(ReadingProgress.Time, load().progress)
    }

    @Test fun oldBarPlusTimeKeepsTheTimeReadout() {
        prefs().putBoolean("show_progress_bar", true).putBoolean("show_book_time_remaining", true).commit()
        assertEquals(ReadingProgress.Time, load().progress)
    }

    @Test fun aSavedProgressChoiceBeatsTheOldBooleans() {
        prefs().putBoolean("show_progress_bar", true).putString("reading_progress", "None").commit()
        assertEquals(ReadingProgress.None, load().progress)
    }

    @Test fun savingWritesTheNewChoiceAndItRoundTrips() {
        prefs().commit()
        val store = EpubReaderPreferencesStore(context, eInk = false)
        store.update { it.copy(progress = ReadingProgress.Time) }
        assertEquals(ReadingProgress.Time, EpubReaderPreferencesStore(context, eInk = false).load().progress)
    }

    @Test fun timeShowsChapterAndBookTogetherBarAndNoneShowNoTimeLabels() {
        val statistics = ReadingStatistics(chapterRemainingMillis = 12 * 60_000L, bookRemainingMillis = 65 * 60_000L)
        assertEquals(listOf("Chapter 12 min", "Book 1 hr 5 min"), persistentReadingLabels(statistics, ReadingProgress.Time))
        assertEquals(listOf("Chapter learning pace", "Book learning pace"), persistentReadingLabels(ReadingStatistics(), ReadingProgress.Time))
        assertTrue(persistentReadingLabels(statistics, ReadingProgress.Bar).isEmpty())
        assertTrue(persistentReadingLabels(statistics, ReadingProgress.None).isEmpty())
        assertFalse(persistentFooterEnabled(ReadingProgress.None))
        assertTrue(persistentFooterEnabled(ReadingProgress.Bar))
        assertTrue(persistentFooterEnabled(ReadingProgress.Time))
    }

    @Test fun anyOldSharpnessStrengthBecomesOn() {
        listOf("Light", "Medium", "Strong", "On").forEach { saved ->
            prefs().putString("text_sharpness", saved).commit()
            assertEquals(saved, TextSharpness.On, load().sharpness)
        }
        prefs().putString("text_sharpness", "Off").commit()
        assertEquals(TextSharpness.Off, load().sharpness)
    }

    @Test fun sharperTextIsOnByDefaultOnlyOnEInkWhenNothingIsSaved() {
        prefs().commit()
        assertEquals(TextSharpness.Off, load(eInk = false).sharpness)
        assertEquals(TextSharpness.On, load(eInk = true).sharpness)
    }

    @Test fun aSavedOffIsNeverOverriddenOnEInk() {
        prefs().putString("text_sharpness", "Off").commit()
        assertEquals(TextSharpness.Off, load(eInk = true).sharpness)
    }

    @Test fun onUsesTheMiddleStrength() {
        assertTrue(sharpTextScript(TextSharpness.On).contains("-webkit-text-stroke:0.2px"))
    }
}
