package net.kdhmedia.mochil.wiki

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WikiResponseErrorTest {
    private fun outcome() = JSONObject().put("error", "Saved; refresh the index.")
        .put("code", "vault_index_refresh_failed").put("committed", true).put("index_refreshed", false)

    @Test fun `only explicit typed server outcome acknowledges a mutation`() {
        val error = decodeWikiResponseError(500, outcome().toString())
        assertTrue(error.mutationCommitted)
        assertEquals(500, error.status)
        assertEquals("Saved; refresh the index.", error.message)
        for ((key, value) in listOf("code" to "unknown", "committed" to "true", "index_refreshed" to "false")) {
            assertFalse(key, decodeWikiResponseError(500, outcome().put(key, value).toString()).mutationCommitted)
        }
        for (key in listOf("code", "committed", "index_refreshed")) {
            val payload = outcome().apply { remove(key) }
            assertFalse(key, decodeWikiResponseError(500, payload.toString()).mutationCommitted)
        }
        assertFalse(decodeWikiResponseError(409, outcome().toString()).mutationCommitted)
    }

    @Test fun `malformed and legacy errors keep safe fallback semantics`() {
        for (text in listOf("", "<html>Sign in</html>", "null", "[]", "{", "{\"error\":true}")) {
            val error = decodeWikiResponseError(500, text)
            assertFalse(error.mutationCommitted)
            assertEquals("The server returned 500.", error.message)
        }
        assertEquals("Conflict", decodeWikiResponseError(409, "{\"error\":\"Conflict\"}").message)
        assertTrue(decodeWikiResponseError(303, "").message!!.contains("Reconnect"))
    }
}
