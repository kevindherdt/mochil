package net.kdhmedia.mochil.readalong

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ReadAlongOverlayTest {
    private val overlay = ReadAlongOverlay.parse(
        JSONObject(
            """{"audio":["Audio/1.mp4","Audio/2.mp4"],"chapters":[
              {"href":"OEBPS/One.html","clips":[["a",0,0,4.0],["b",0,4.0,8.0],["c",0,8.0,12.0],["d",0,12.0,16.0]]},
              {"href":"OEBPS/Two%20.html","clips":[["e",1,0,5.0],["f",1,5.0,200.0]]}]}""",
        ),
    )!!

    @Test fun parsesClipsInReadingOrderWithTheirAudioFile() {
        assertEquals(listOf("a", "b", "c", "d", "e", "f"), overlay.clips.map { it.id })
        assertEquals(listOf("Audio/1.mp4", "Audio/2.mp4"), overlay.audio)
        assertEquals(1, overlay.clips[4].audio)
    }

    @Test fun aBookWithoutNarrationHasNoOverlay() {
        assertNull(ReadAlongOverlay.parse(JSONObject("""{"audio":[],"chapters":[]}""")))
        assertNull(ReadAlongOverlay.parse(JSONObject("""{"audio":["x.mp4"],"chapters":[{"href":"c.html","clips":[["a",3,0,1]]}]}""")))
    }

    @Test fun startsAtTheShareOfTheChapterThePageIsAt() {
        assertEquals(0, overlay.startClip("OEBPS/One.html", 0.0))
        assertEquals(2, overlay.startClip("/OEBPS/One.html#x", 0.5))
        assertEquals(3, overlay.startClip("OEBPS/One.html", 1.0))
        assertEquals(4, overlay.startClip("OEBPS/Two .html", Double.NaN)) // escapes and the leading slash do not matter
        assertEquals(-1, overlay.startClip("OEBPS/Cover.html", 0.0))
    }

    @Test fun findsTheSpokenClipFromTheAudioPosition() {
        assertEquals(1, overlay.clipAt(0, 5.0, hint = 0))
        assertEquals(3, overlay.clipAt(0, 13.0, hint = 0))
        assertEquals(0, overlay.clipAt(0, 1.0, hint = 3), "going back finds the earlier clip")
        assertEquals(4, overlay.clipAt(1, 0.5, hint = 0), "the same time in another file is another clip")
        assertEquals(-1, overlay.clipAt(0, 99.0), "past the last clip of the file")
    }

    @Test fun aVeryLongClipIsNotOneSentence() {
        val long = overlay.clips[5]
        assertTrue(long.end - long.begin > ReadAlongOverlay.MAX_SENTENCE_SECONDS)
        assertNotNull(overlay.clips[4])
    }

    @Test fun remembersHowFarThroughTheChapterAClipIs() {
        assertEquals(0.0, overlay.progressionOf(0), 0.0)
        assertEquals(0.5, overlay.progressionOf(2), 0.0)
        assertEquals(0.5, overlay.progressionOf(5), 0.0)
    }

    @Test fun aPageWithNoNarrationStartsAtTheNextChapterThatHasSome() {
        val order = listOf("OEBPS/Cover.html", "OEBPS/Title.html", "OEBPS/One.html", "OEBPS/Two .html")
        assertEquals(0, overlay.startClipFrom(order, "OEBPS/Cover.html", 0.0))
        assertEquals(4, overlay.startClipFrom(order.reversed().drop(0), "OEBPS/Two .html", 0.0))
        assertEquals(0, overlay.startClipFrom(listOf("OEBPS/One.html"), "OEBPS/Unknown.html", 0.0), "an unknown page starts at the beginning, not at random")
    }

    @Test fun listsTheClipsOfOneChapterInOrder() {
        assertEquals(listOf(0, 1, 2, 3), overlay.clipsOf("OEBPS/One.html#top"))
        assertEquals(emptyList<Int>(), overlay.clipsOf("OEBPS/Nothing.html"))
    }

    private fun <T> assertEquals(expected: List<T>, actual: List<T>) = org.junit.Assert.assertEquals(expected, actual)
    private fun assertEquals(expected: Int, actual: Int, message: String) = org.junit.Assert.assertEquals(message, expected, actual)
}
