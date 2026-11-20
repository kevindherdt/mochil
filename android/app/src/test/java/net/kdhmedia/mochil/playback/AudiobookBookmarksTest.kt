package net.kdhmedia.mochil.playback

import net.kdhmedia.mochil.audio.AudioChapter
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class AudiobookBookmarksTest {
    private val chapters = listOf(
        AudioChapter("Opening", 0.0, 60.0),
        AudioChapter("The road", 60.0, 120.0),
        AudioChapter("Arrival", 120.0, 0.0),
    )

    @Test fun chapterLookupUsesStartInclusiveEndExclusiveRanges() {
        assertEquals("Opening", chapterAt(chapters, 0)?.title)
        assertEquals("The road", chapterAt(chapters, 60_000)?.title)
        assertEquals("Arrival", chapterAt(chapters, 900_000)?.title)
    }

    @Test fun chapterLookupRejectsPositionsBeforeFirstChapter() {
        assertNull(chapterAt(listOf(AudioChapter("Later", 10.0, 20.0)), 5_000))
    }
}
