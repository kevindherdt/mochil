package net.kdhmedia.mochil.readalong

import net.kdhmedia.mochil.catalogue.toBookSummary
import org.json.JSONObject
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ReadAlongFlagTest {
    private fun book(extra: String) = JSONObject("""{"id":"b","library_id":"l","title":"Neuromancer","authors":["W. Gibson"]$extra}""").toBookSummary()

    @Test fun aBookWithItsOwnRecordingIsMarked() = assertTrue(book(""","read_along":true""").readAlong)
    @Test fun anotherBookIsNot() = assertFalse(book("").readAlong)
}
