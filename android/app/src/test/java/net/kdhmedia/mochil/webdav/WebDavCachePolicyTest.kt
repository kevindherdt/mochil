package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Test
import org.json.JSONArray
import org.json.JSONObject

class WebDavCachePolicyTest {
    @Test fun credentialIdentityInvalidatesEveryDerivedCache() {
        val first = WebDavConnection("one", "Cloud", "https://cloud.example/books/", "reader", "secret")
        assertEquals(webDavCredentialKey(first), webDavCredentialKey(first.copy(name = "Renamed")))
        assertNotEquals(webDavCredentialKey(first), webDavCredentialKey(first.copy(password = "changed", cacheGeneration = "rotated")))
        assertNotEquals(webDavCredentialKey(first), webDavCredentialKey(first.copy(rootUrl = "https://cloud.example/other/", cacheGeneration = "rotated")))
        assertFalse(webDavCredentialKey(first).contains("secret"))
    }

    @Test fun legacyPageCacheWithoutGenerationCannotMatchCurrentCredentials() {
        val legacy = JSONArray().put(JSONObject().put("connection", "one").put("url", "https://cloud.example/books/")
            .put("entries", JSONArray()).put("used", 1)).toString().encodeToByteArray()
        val decoded = decodeWebDavCache(legacy).single()
        val current = WebDavConnection("one", "Cloud", "https://cloud.example/books/", "reader", "secret", "generation")
        assertEquals("", decoded.credentialKey)
        assertNotEquals(webDavCredentialKey(current), decoded.credentialKey)
    }

    @Test fun notModifiedResponseRetainsLastKnownGoodEntries() {
        val cached = WebDavPage("https://cloud.example/books/", listOf(
            WebDavEntry("https://cloud.example/books/A.epub", "A.epub", false, 42, "application/epub+zip", "item-tag", "today"),
        ), "folder-tag")
        val result = resolveWebDavPage(cached, WebDavPage(cached.url, emptyList(), "folder-tag", notModified = true))
        assertEquals(cached, result)
    }

    @Test fun notModifiedWithoutCacheBecomesAnEmptyUsablePage() {
        val fetched = WebDavPage("https://cloud.example/books/", emptyList(), "folder-tag", notModified = true)
        val result = resolveWebDavPage(null, fetched)
        assertFalse(result.notModified)
        assertEquals(fetched.url, result.url)
    }
}
