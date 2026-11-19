package net.kdhmedia.mochil.opds

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class OpdsConnectionStoreTest {
    @Test fun identityIsStableButSeparatesAccounts() {
        val first = stableConnectionId("https://books.example/opds", "alice")
        assertEquals(first, stableConnectionId("https://books.example/opds", "alice"))
        assertNotEquals(first, stableConnectionId("https://books.example/opds", "bob"))
    }

    @Test fun editingAddressAndNamePreservesConnectionIdentity() {
        val original = OpdsConnection("stable", "Books", "https://old.example/opds", "alice", "secret", "generation")
        val updated = updatedOpdsConnection(original, "New books", "https://new.example/opds", "alice", "secret")
        assertEquals("stable", updated.id)
        assertEquals("New books", updated.name)
        assertEquals("https://new.example/opds", updated.catalogueUrl)
        assertNotEquals(original.cacheGeneration, updated.cacheGeneration)
    }

    @Test fun simpleRenamePreservesArtworkCacheGeneration() {
        val original = OpdsConnection("stable", "Books", "https://books.example/opds", "alice", "secret", "generation")
        val updated = updatedOpdsConnection(original, "Renamed", original.catalogueUrl, original.username, original.password)
        assertEquals("generation", updated.cacheGeneration)
    }
}
