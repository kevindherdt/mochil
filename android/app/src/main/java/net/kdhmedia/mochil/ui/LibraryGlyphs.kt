package net.kdhmedia.mochil.ui

import androidx.compose.ui.graphics.vector.ImageVector
import net.kdhmedia.mochil.ui.kit.LucideIcons

private val AUDIO_ID = Regex("audiobook|storyteller|podcast|audio")
private val VIDEO_ID = Regex("movie|series|tv|film|anime")
private val COMIC_ID = Regex("manga|comic")
private val MUSIC_ID = Regex("music|album|song")
private val TEXT_ID = Regex("test|text|scripture|doc|paper")

/**
 * The Lucide glyph a library wears, as web/src/features/library/kinds.ts `libraryIcon` picks it. One kind can carry several
 * libraries (this server reports "books" for Light Novels, Storyteller and the test shelf alike), so the id is read first, in
 * the web's order, and the kind decides only when the id says nothing.
 */
internal fun libraryGlyphId(kind: String, id: String): String {
    val k = kind.lowercase()
    val i = id.lowercase()
    return when {
        AUDIO_ID.containsMatchIn(i) -> if (k == "podcasts") "mic" else "headphones"
        VIDEO_ID.containsMatchIn(i) -> "clapperboard"
        COMIC_ID.containsMatchIn(i) -> "book-image"
        MUSIC_ID.containsMatchIn(i) -> "eighth-note"
        TEXT_ID.containsMatchIn(i) -> "file-text"
        else -> kindGlyphId(k)
    }
}

private fun kindGlyphId(kind: String): String = when (kind) {
    "audio", "audiobooks" -> "headphones"
    "video", "movies", "tv", "anime" -> "clapperboard"
    "manga", "comics" -> "book-image"
    "documents" -> "file-text"
    "music" -> "eighth-note"
    "podcasts" -> "mic"
    else -> "book-open"
}

/** The glyph itself, at Lucide's own weight. Build it once per library list (`remember`), not per frame. */
internal fun libraryGlyph(kind: String, id: String): ImageVector = LucideIcons.withStroke(libraryGlyphId(kind, id), 2f)
