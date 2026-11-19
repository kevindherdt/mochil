package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class LibraryDirectoryCodecTest {
    @Test fun cacheRoundTripPreservesDisplayIdentity() {
        val libraries = listOf(
            LibrarySummary("books", "Books", "books", 324),
            LibrarySummary("light-novels", "Light Novels", "lightnovels", 5_149),
        )

        assertEquals(libraries, LibraryDirectoryCodec.decode(LibraryDirectoryCodec.encode(libraries)))
    }

    @Test fun malformedOrStructurallyInvalidCacheFailsClosed() {
        assertTrue(LibraryDirectoryCodec.decode("not json").isEmpty())
        assertTrue(LibraryDirectoryCodec.decode("""[{"id":"","name":"Missing identity"}]""").isEmpty())
        assertTrue(LibraryDirectoryCodec.decode("""[{"id":"books","name":""}]""").isEmpty())
    }
}
