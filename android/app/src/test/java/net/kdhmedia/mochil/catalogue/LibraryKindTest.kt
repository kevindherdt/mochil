package net.kdhmedia.mochil.catalogue

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * MediaKindStyle.kt, TvNavRail.kt and TvConnectedApp.kt each used to carry their own
 * `when (kind: String)` list of which kinds count as "a publication library" -- the exact kind of
 * list that drifts. This pins manga and comics (the audit's flagged disagreement) and every other
 * kind to one canonical grouping, matching the server's `isPublicationKind` in
 * internal/appconfig/config.go.
 */
class LibraryKindTest {
    @Test fun `manga and comics agree on category with books`() {
        assertEquals(LibraryCategory.Publication, LibraryKind.fromWire("manga").category)
        assertEquals(LibraryCategory.Publication, LibraryKind.fromWire("comics").category)
        assertEquals(LibraryKind.fromWire("manga").category, LibraryKind.fromWire("comics").category)
        assertEquals(LibraryKind.fromWire("books").category, LibraryKind.fromWire("manga").category)
        assertEquals(LibraryKind.fromWire("books").category, LibraryKind.fromWire("comics").category)
    }

    @Test fun `publication kinds match the server's isPublicationKind switch`() {
        // internal/appconfig/config.go#isPublicationKind: case "books", "comics", "manga", "mixed"
        listOf("books", "comics", "manga", "mixed").forEach {
            assertTrue(it, LibraryKind.fromWire(it).isPublication())
        }
        listOf("movies", "tv", "anime", "music", "audiobooks", "podcasts", "vault").forEach {
            assertTrue(it, !LibraryKind.fromWire(it).isPublication())
        }
    }

    @Test fun `legacy spellings normalize to the same kind`() {
        assertEquals(LibraryKind.Comics, LibraryKind.fromWire("comic"))
        assertEquals(LibraryKind.LightNovels, LibraryKind.fromWire("light-novels"))
        assertEquals(LibraryKind.Tv, LibraryKind.fromWire("TV"))
    }

    @Test fun `every kind that groups with books on screen agrees it is a publication`() {
        // TvNavRail.tvRailIcon, TvBrowseModes.tvBrowseModes, TvLibraryScreen.isPublicationLibraryForTv and
        // MobileLibraryRoute all list "books, lightnovels, manga, comics, mixed, calibre" together.
        listOf("books", "lightnovels", "manga", "comics", "mixed", "calibre").forEach {
            assertTrue(it, LibraryKind.fromWire(it).isPublication())
        }
    }
}
