package net.kdhmedia.mochil.opds

import java.net.URI
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class OpdsArtworkRepositoryTest {
    @Test fun artworkIdentityIsStableAndDoesNotContainTheUrl() {
        val url = "https://books.example/private/cover.jpg"
        val connection = OpdsConnection("one", "Books", "https://books.example/opds", "reader", "secret")
        assertEquals(artworkCacheKey(connection, url), artworkCacheKey(connection, url))
        assertFalse(artworkCacheKey(connection, url).contains("books"))
        assertFalse(artworkCacheKey(connection, url).contains("secret"))
    }

    @Test fun artworkIdentityChangesWithAuthenticatedSourceIdentity() {
        val url = "https://books.example/private/cover.jpg"
        val first = OpdsConnection("one", "Books", "https://books.example/opds", "reader", "secret")
        assertFalse(artworkCacheKey(first, url) == artworkCacheKey(first.copy(password = "changed", cacheGeneration = "rotated"), url))
        assertFalse(artworkCacheKey(first, url) == artworkCacheKey(first.copy(id = "another"), url))
    }

    @Test fun credentialsAreEligibleOnlyForTheExactCatalogueOrigin() {
        val root = URI("https://books.example/opds")
        assertTrue(sameArtworkOrigin(root, URI("https://books.example/covers/one.jpg")))
        assertFalse(sameArtworkOrigin(root, URI("https://cdn.example/covers/one.jpg")))
        assertFalse(sameArtworkOrigin(root, URI("https://books.example:8443/covers/one.jpg")))
    }

    @Test fun basicHeaderEncodesCredentialsWithoutPlacingThemInAUrl() {
        val connection = OpdsConnection("one", "Books", "https://books.example/opds", "reader", "secret")
        assertEquals("Basic cmVhZGVyOnNlY3JldA==", basicOpdsAuthorization(connection))
    }
}
