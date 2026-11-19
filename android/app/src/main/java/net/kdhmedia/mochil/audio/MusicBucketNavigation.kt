package net.kdhmedia.mochil.audio

internal data class MusicBucketJump(val state: MusicState, val offset: Int)

internal fun MusicState.supportsMusicBucketNavigation(): Boolean =
    libraryKind == "music" && libraryId.isNotBlank() && selectedArtist == null && selectedGroup == null &&
        !downloadedOnly && !favoritesOnly && !offlineOnly &&
        grouping in setOf(MusicGrouping.Tracks, MusicGrouping.Albums, MusicGrouping.Artists)

/** Server offsets describe the queried root, never an album or local-only projection. */
internal fun MusicState.musicBucketJump(requested: String): MusicBucketJump? {
    if (!supportsMusicBucketNavigation()) return null
    val target = navigation.firstOrNull { it.bucket == requested && it.offset >= 0 } ?: return null
    val clear = bucket == requested
    return MusicBucketJump(copy(
        groups = emptyList(), tracks = emptyList(), nextCursor = null, nextOffset = null,
        bucket = if (clear) "" else requested, loading = true, loadingMore = false, error = null,
    ), if (clear) 0 else target.offset)
}
