package net.kdhmedia.mochil.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/** Which glyph a library's pill and badge wear: web/src/features/library/kinds.ts `libraryIcon`, id patterns first, then kind. */
class LibraryGlyphsTest {
    @Test fun `the kind picks the glyph when the id says nothing`() {
        assertEquals("book-open", libraryGlyphId("books", "shelf"))
        assertEquals("headphones", libraryGlyphId("audiobooks", "shelf"))
        assertEquals("headphones", libraryGlyphId("audio", "shelf"))
        assertEquals("clapperboard", libraryGlyphId("movies", "shelf"))
        assertEquals("clapperboard", libraryGlyphId("tv", "shelf"))
        assertEquals("clapperboard", libraryGlyphId("anime", "shelf"))
        assertEquals("clapperboard", libraryGlyphId("video", "shelf"))
        assertEquals("book-image", libraryGlyphId("manga", "shelf"))
        assertEquals("book-image", libraryGlyphId("comics", "shelf"))
        assertEquals("file-text", libraryGlyphId("documents", "shelf"))
        assertEquals("eighth-note", libraryGlyphId("music", "shelf"))
        assertEquals("mic", libraryGlyphId("podcasts", "shelf"))
    }

    @Test fun `an unknown kind is an open book, in any case`() {
        assertEquals("book-open", libraryGlyphId("somethingnew", "shelf"))
        assertEquals("clapperboard", libraryGlyphId("MOVIES", "shelf"))
    }

    @Test fun `one kind can carry several libraries, so the id tells them apart`() {
        // This server reports kind "books" for Light Novels, Storyteller and the test shelf alike.
        assertEquals("headphones", libraryGlyphId("books", "storyteller"))
        assertEquals("headphones", libraryGlyphId("books", "audiobooks-2"))
        assertEquals("mic", libraryGlyphId("podcasts", "podcast-feed"))
        assertEquals("clapperboard", libraryGlyphId("books", "tv-series"))
        assertEquals("book-image", libraryGlyphId("books", "manga"))
        assertEquals("eighth-note", libraryGlyphId("books", "music-library"))
        assertEquals("file-text", libraryGlyphId("books", "documents"))
        assertEquals("file-text", libraryGlyphId("books", "test"))
        assertEquals("book-open", libraryGlyphId("books", "lightnovels"))
    }

    @Test fun `the audio patterns win over the video ones, as on the web`() {
        // "audio" is tested before "tv"/"series", so a shelf id that has both is an audio shelf.
        assertEquals("headphones", libraryGlyphId("books", "audio-series"))
    }
}
