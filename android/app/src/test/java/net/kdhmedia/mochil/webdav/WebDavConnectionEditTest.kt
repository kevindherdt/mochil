package net.kdhmedia.mochil.webdav

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

class WebDavConnectionEditTest {
    @Test fun editingAddressAndNamePreservesConnectionIdentity() {
        val original = WebDavConnection("stable", "Cloud", "https://old.example/dav", "alice", "secret", "generation")
        val updated = updatedWebDavConnection(original, "Archive", "https://new.example/dav", "alice", "secret")
        assertEquals("stable", updated.id)
        assertEquals("Archive", updated.name)
        assertEquals("https://new.example/dav", updated.rootUrl)
        assertNotEquals(original.cacheGeneration, updated.cacheGeneration)
    }

    @Test fun simpleRenamePreservesCacheGeneration() {
        val original = WebDavConnection("stable", "Cloud", "https://cloud.example/dav", "alice", "secret", "generation")
        val updated = updatedWebDavConnection(original, "Archive", original.rootUrl, original.username, original.password)
        assertEquals("generation", updated.cacheGeneration)
    }
}
