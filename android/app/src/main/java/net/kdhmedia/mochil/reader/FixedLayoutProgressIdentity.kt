package net.kdhmedia.mochil.reader

internal data class FixedLayoutProgressIdentity(val libraryId: String, val bookId: String)

internal fun fixedLayoutStorageKey(libraryId: String, bookId: String): String =
    "${libraryId.length}:$libraryId${bookId.length}:$bookId"

/**
 * New server intents carry both IDs explicitly. The combined key fallback keeps
 * older downloads readable, but local-only publications must never call server
 * progress endpoints.
 */
internal fun fixedLayoutProgressIdentity(
    explicitLibraryId: String,
    explicitBookId: String,
    legacyProgressKey: String,
): FixedLayoutProgressIdentity? {
    if (explicitLibraryId.isNotBlank() && explicitBookId.isNotBlank()) {
        return FixedLayoutProgressIdentity(explicitLibraryId, explicitBookId)
    }
    val separator = legacyProgressKey.indexOf(':')
    if (separator <= 0 || separator == legacyProgressKey.lastIndex) return null
    val libraryId = legacyProgressKey.substring(0, separator)
    if (libraryId == "local" || libraryId == "webdav") return null
    return FixedLayoutProgressIdentity(libraryId, legacyProgressKey.substring(separator + 1))
}
