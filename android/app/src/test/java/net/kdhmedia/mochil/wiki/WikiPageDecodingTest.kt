package net.kdhmedia.mochil.wiki

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class WikiPageDecodingTest {
    private fun page() = JSONObject("""{"id":"note","content":"","writable":true,"edit_revision":"r1"}""")

    @Test fun `explicit empty content remains a valid editable page`() {
        val result = decodeWikiPage(page(), "note")
        assertEquals("", result.content)
        assertTrue(result.writable)
        assertEquals("r1", result.revision)
    }

    @Test fun `missing or non text content cannot become an empty page`() {
        for (invalid in listOf<Any>(JSONObject.NULL, 12, true, JSONObject())) {
            assertTrue(runCatching { decodeWikiPage(page().put("content", invalid), "note") }.isFailure)
        }
        assertTrue(runCatching { decodeWikiPage(page().apply { remove("content") }, "note") }.isFailure)
        assertTrue(runCatching { decodeWikiPage(page(), "another") }.isFailure)
    }

    @Test fun `editable page requires a revision and actual boolean permission`() {
        for (revision in listOf("", " ", JSONObject.NULL, 1)) {
            assertTrue(runCatching { decodeWikiPage(page().put("edit_revision", revision), "note") }.isFailure)
        }
        assertTrue(runCatching { decodeWikiPage(page().put("writable", "true"), "note") }.isFailure)
        val readOnly = page().apply { remove("writable"); remove("edit_revision") }
        assertFalse(decodeWikiPage(readOnly, "note").writable)
    }
}
