package net.kdhmedia.mochil.catalogue

/**
 * The server's library "kind" values (see `internal/appconfig/config.go`'s `isPublicationKind`/`Validate`
 * switches) normalized into one type, so every screen that groups libraries by what they contain -- the
 * TV rail icon, the per-kind colour/label, the resume-refresh policy, the publication-grouping checks --
 * agrees on the grouping. Before this type existed, three or more independent `when (kind: String)` maps
 * each repeated their own manga/comics/books membership list by hand, which is exactly the kind of drift
 * the server's own `isPublicationKind` (manga and comics are both publication libraries, same as books,
 * lightnovels, mixed and calibre) is the one source of truth for.
 */
internal enum class LibraryCategory { Publication, Video, Audio, Podcast, Document }

internal enum class LibraryKind(val wire: String, val category: LibraryCategory) {
    Books("books", LibraryCategory.Publication),
    LightNovels("lightnovels", LibraryCategory.Publication),
    Manga("manga", LibraryCategory.Publication),
    Comics("comics", LibraryCategory.Publication),
    Mixed("mixed", LibraryCategory.Publication),
    Calibre("calibre", LibraryCategory.Publication),
    Movies("movies", LibraryCategory.Video),
    Tv("tv", LibraryCategory.Video),
    Anime("anime", LibraryCategory.Video),
    Music("music", LibraryCategory.Audio),
    Audiobooks("audiobooks", LibraryCategory.Audio),
    Podcasts("podcasts", LibraryCategory.Podcast),
    Vault("vault", LibraryCategory.Document);

    companion object {
        /**
         * Accepts the server's wire value, plus the legacy singular/hyphenated spellings some screens used
         * to normalize by hand, and falls back to [Books] (a publication library) for anything unrecognised
         * so an unknown kind still reads as "a shelf of things to read" rather than vanishing from the UI.
         */
        fun fromWire(kind: String): LibraryKind {
            val normalized = kind.trim().lowercase().replace("-", "").replace("_", "").replace(" ", "")
            return when (normalized) {
                "book", "books" -> Books
                "lightnovel", "lightnovels", "novel", "novels" -> LightNovels
                "manga" -> Manga
                "comic", "comics" -> Comics
                "mixed" -> Mixed
                "calibre" -> Calibre
                "movie", "movies" -> Movies
                "tv", "television", "tvshow", "tvshows" -> Tv
                "anime", "video", "videos" -> Anime
                "music" -> Music
                "audiobook", "audiobooks" -> Audiobooks
                "podcast", "podcasts" -> Podcasts
                "vault" -> Vault
                else -> Books
            }
        }
    }
}

/** True for the same kinds the server's `isPublicationKind` validates against (plus the client-only
 * lightnovels/calibre aliases every publication screen already treats as books). */
internal fun LibraryKind.isPublication(): Boolean = category == LibraryCategory.Publication
