package net.kdhmedia.mochil.ui

import net.kdhmedia.mochil.catalogue.LibraryCategory
import net.kdhmedia.mochil.catalogue.LibraryKind
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * MediaKindStyle.kt, TvNavRail.kt and TvConnectedApp.kt each used to carry their own hand-written
 * `when (kind: String)` list of which kinds belong with "books" (manga, comics, lightnovels, mixed,
 * calibre). Now all three parse through the one [LibraryKind] type, so they can no longer disagree on
 * whether manga and comics group with books or with something else.
 */
class MediaKindGroupingTest {
    @Test fun `TV rail icon groups manga and comics with books, same as MediaKindStyle's category`() {
        assertEquals(tvRailIcon("manga"), tvRailIcon("comics"))
        assertEquals(tvRailIcon("manga"), tvRailIcon("books"))
        assertEquals(LibraryCategory.Publication, LibraryKind.fromWire("manga").category)
        assertEquals(LibraryCategory.Publication, LibraryKind.fromWire("comics").category)
    }

    @Test fun `TV resume refresh treats manga and comics the same as books, not like video`() {
        assertEquals(tvResumeRefreshes("Library", "manga"), tvResumeRefreshes("Library", "comics"))
        assertEquals(tvResumeRefreshes("Library", "manga"), tvResumeRefreshes("Library", "books"))
        assertEquals(setOf(TvResumeRefresh.Libraries), tvResumeRefreshes("Library", "manga"))
    }

    @Test fun `MediaKindStyle still gives manga and comics their own icon but the same publication category`() {
        assertEquals("Manga", mediaKindStyle("manga").label)
        assertEquals("Comic", mediaKindStyle("comics").label)
        assertEquals(LibraryKind.fromWire("manga").category, LibraryKind.fromWire("comics").category)
    }
}
