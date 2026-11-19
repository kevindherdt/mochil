package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

class TvTitlesTest {
    @Test fun releaseLeftoversAreDropped() {
        assertEquals("The Witch - Part 1 The Subversion (2018)", tvCleanTitle("The Witch - Part 1 The Subversion (2018) Bluray-1080p"))
        assertEquals("Extracurricular - S01E01 - Episode 1", tvCleanTitle("Extracurricular - S01E01 - Episode 1 WEBDL-2160p"))
        assertEquals("Baki Hanma (2021)", tvCleanTitle("Baki Hanma (2021) [tvdbid=362999]"))
        assertEquals("The World of Kanako (2014)", tvCleanTitle("The World of Kanako (2014) Bluray-1080p"))
    }

    @Test fun plainTitlesAndEmptyResultsSurvive() {
        assertEquals("Neuromancer", tvCleanTitle("Neuromancer"))
        assertEquals("1080p", tvCleanTitle("1080p"))
    }

    @Test fun countsAgree() {
        assertEquals("1 episode", tvCount(1, "episode"))
        assertEquals("26 episodes", tvCount(26, "episode"))
    }
}
