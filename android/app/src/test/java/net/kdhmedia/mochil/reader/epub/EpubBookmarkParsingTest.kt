package net.kdhmedia.mochil.reader.epub

import org.junit.Assert.assertEquals
import org.junit.Test

class EpubBookmarkParsingTest {
    private val decode = { value: org.json.JSONObject -> value.getString("name") }

    @Test fun oneUnreadableBookmarkDoesNotHideTheOthers() {
        val read = readSavedEntries("""[{"name":"a"},{"nothing":true},{"name":"b"}]""", decode)
        assertEquals(listOf("a", "b"), read)
    }

    @Test fun unreadableTextGivesNoBookmarks() {
        assertEquals(0, readSavedEntries("not json", decode).size)
    }
}
