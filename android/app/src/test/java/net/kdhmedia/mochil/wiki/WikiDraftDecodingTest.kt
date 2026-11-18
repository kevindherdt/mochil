package net.kdhmedia.mochil.wiki

import org.json.JSONObject
import org.junit.Assert.*
import org.junit.Test

class WikiDraftDecodingTest {
    private fun payload(content: String) = JSONObject()
        .put("library_id", "vault").put("page_id", "page")
        .put("revision", "revision").put("content", content).put("updated_at", 123L)

    @Test fun `recovery preserves exact text including empty drafts and line endings`() {
        for (content in listOf("", "---\r\ntitle: Notes\r\n---\r\n日本語 🎵\r\n", "x".repeat(2_000_000))) {
            val draft = decodeWikiDraft(payload(content))
            assertEquals(content, draft.content)
            assertEquals("revision", draft.revision)
            assertEquals(123L, draft.updatedAt)
            assertTrue(draft.isValidRecoveryDraft())
        }
    }

    @Test fun `oversized recovery is rejected rather than truncated`() {
        val value = payload("x".repeat(2_000_001))
        assertTrue(runCatching { decodeWikiDraft(value) }.isFailure)
        assertEquals(2_000_001, value.getString("content").length)
    }

    @Test fun `recovery requires both library and page identity`() {
        for (field in listOf("library_id", "page_id")) {
            assertTrue(runCatching { decodeWikiDraft(payload("text").put(field, " ")) }.isFailure)
            val missing = payload("text").apply { remove(field) }
            assertTrue(runCatching { decodeWikiDraft(missing) }.isFailure)
        }
    }
}
