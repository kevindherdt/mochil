package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.video.VideoTitleDetails
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class TvTitleFactsTest {
    @Test
    fun theLineJoinsYearRuntimeAndGenres() {
        assertEquals("2006 · 24m · Mystery, Horror", tvFactsLine(2006, VideoTitleDetails(listOf("Mystery", "Horror"), 24, 7.8, "TV-MA")))
        assertEquals("1999 · 2h 16m", tvFactsLine(1999, VideoTitleDetails(runtimeMinutes = 136)))
        assertEquals("2h", tvFactsLine(0, VideoTitleDetails(runtimeMinutes = 120)))
        assertEquals("2006", tvFactsLine(2006, null))
    }

    @Test
    fun theScoreShowsOneDecimalOnlyWhenThereIsOne() {
        assertEquals("★ 7.8", tvScoreLabel(VideoTitleDetails(rating = 7.8)))
        assertNull(tvScoreLabel(VideoTitleDetails()))
    }
}
