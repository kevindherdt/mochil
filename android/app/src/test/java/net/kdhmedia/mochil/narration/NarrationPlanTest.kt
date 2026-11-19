package net.kdhmedia.mochil.narration

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class NarrationPlanTest {
    private val chapters = listOf(
        NarrationChapter(0, "cover.xhtml", "Cover", matter = false),
        NarrationChapter(1, "OEBPS/Text/copyright.xhtml", "Copyright", matter = true),
        NarrationChapter(2, "OEBPS/Text/chapter%201.xhtml", "Chapter 1", matter = false),
        NarrationChapter(3, "OEBPS/Text/about.xhtml", "About the Author", matter = true),
        NarrationChapter(4, "OEBPS/Text/chapter2.xhtml", "Chapter 2", matter = false),
        NarrationChapter(5, "OEBPS/Text/also-by.xhtml", "Also by", matter = true),
    )

    @Test fun readerResourceIsMatchedToTheServersChapterWhateverTheEscapingOrFragment() {
        assertEquals(2, NarrationPlan.chapterFor("OEBPS/Text/chapter 1.xhtml", chapters)?.index)
        assertEquals(2, NarrationPlan.chapterFor("/OEBPS/Text/chapter%201.xhtml#p3", chapters)?.index)
        assertEquals(4, NarrationPlan.chapterFor("OEBPS/Text/chapter2.xhtml", chapters)?.index)
        assertNull(NarrationPlan.chapterFor("OEBPS/Text/missing.xhtml", chapters))
    }

    @Test fun readingStartsAtTheFirstSentenceOnOrAfterThePage() {
        val sentences = listOf(0.0, 0.1, 0.25, 0.5, 0.9).map { NarrationSentence("s", it, "", "") }
        assertEquals(0, NarrationPlan.startSentence(sentences, 0.0))
        assertEquals(2, NarrationPlan.startSentence(sentences, 0.2))
        assertEquals(2, NarrationPlan.startSentence(sentences, 0.25))
        assertEquals(4, NarrationPlan.startSentence(sentences, 0.95)) // past the last: the last one
        assertEquals(0, NarrationPlan.startSentence(sentences, Double.NaN))
        assertEquals(0, NarrationPlan.startSentence(emptyList(), 0.4))
    }

    @Test fun theNextChapterSkipsFrontAndBackMatter() {
        assertEquals(2, NarrationPlan.nextChapter(chapters, 0)?.index) // copyright is skipped
        assertEquals(4, NarrationPlan.nextChapter(chapters, 2)?.index) // so is "about the author"
        assertNull(NarrationPlan.nextChapter(chapters, 4)) // only "also by" is left: the book is done
        assertNull(NarrationPlan.nextChapter(emptyList(), 0))
    }

    @Test fun theVoiceFadesOverTheLastTenSecondsOfASleepTimer() {
        assertEquals(1f, sleepFadeVolume(60_000), 0f)
        assertEquals(1f, sleepFadeVolume(10_000), 0f)
        assertEquals(0.5f, sleepFadeVolume(5_000), 0.001f)
        assertEquals(0f, sleepFadeVolume(0), 0f)
        assertEquals(0f, sleepFadeVolume(-5), 0f)
    }

    @Test fun speedsStayWithinWhatTheVoiceCanKeepUpWith() {
        assertTrue(narrationSpeeds.max() <= 1.5f)
        assertTrue(1f in narrationSpeeds)
    }
}

class NarrationModelsTest {
    @Test fun statusListsVoicesAndIsNullWhenNarrationIsNotAvailable() {
        val json = JSONObject("""{"available":true,"voice":"alba","voices":[{"id":"abc","name":"Mine","language":"dutch","custom":true},{"id":"alba","name":"Alba","language":"english"},{"id":"","name":"broken"}]}""")
        val status = parseNarrationStatus(json)!!
        assertEquals("alba", status.defaultVoice)
        assertEquals(listOf("abc", "alba"), status.voices.map { it.id })
        assertTrue(status.voices[0].custom)
        assertEquals("dutch", status.voices[0].language)
        assertNull(parseNarrationStatus(JSONObject("""{"available":false,"voices":[]}""")))
    }

    @Test fun chaptersAndSentencesAreReadFromTheServersAnswer() {
        val chapters = parseNarrationChapters(JSONObject("""{"title":"B","chapters":[{"index":0,"href":"a.xhtml","title":"One"},{"index":1,"href":"b.xhtml","matter":true}]}"""))
        assertEquals(listOf("One", ""), chapters.map { it.title })
        assertEquals(listOf(false, true), chapters.map { it.matter })
        val sentences = parseNarrationSentences(JSONObject("""{"chapter":0,"sentences":[{"text":"Hello there.","progression":0.5,"before":"x","after":"y"},{"text":"  "},{"text":"Again.","progression":7}]}"""))
        assertEquals(listOf("Hello there.", "Again."), sentences.map { it.text })
        assertEquals(1.0, sentences[1].progression, 0.0) // kept within the chapter
        assertEquals("x", sentences[0].before)
    }
}
