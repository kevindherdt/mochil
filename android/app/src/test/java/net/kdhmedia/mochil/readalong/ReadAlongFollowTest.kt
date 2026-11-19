package net.kdhmedia.mochil.readalong

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Test

class ReadAlongFollowTest {
    private val overlay = ReadAlongOverlay.parse(
        JSONObject("""{"audio":["a.mp4"],"chapters":[{"href":"c1.html","clips":[["s0",0,0,3.0],["s1",0,3.0,6.0],["s2",0,6.0,9.0]]},{"href":"c2.html","clips":[["t0",0,9.0,12.0]]}]}"""),
    )!!

    @Test fun readingFromASelectedSentenceStartsAtThatSentenceNotTheChapterStart() {
        val own = overlay.clipsOf("c1.html")
        val ids = own.map { overlay.clips[it].id }
        val picked = own[ids.indexOf("s2")]
        assertEquals(2, picked)
        assertEquals(6.0, overlay.clips[picked].begin, 0.0)
    }

    @Test fun aSelectionOutsideAnyNarratedSentenceFallsBackToThePage() {
        val own = overlay.clipsOf("c1.html")
        val picked = own.getOrNull(-1)
        assertEquals(null, picked)
        assertEquals(0, overlay.startClipFrom(listOf("c1.html", "c2.html"), "c1.html", 0.0))
    }
}
