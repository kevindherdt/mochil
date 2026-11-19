package net.kdhmedia.mochil.audio

/** Immutable state consumed by the mobile and TV music surfaces. */
data class MusicState(
    val libraryId: String = "",
    val libraryKind: String = "music",
    val grouping: MusicGrouping = MusicGrouping.Discover,
    val layout: MusicLayout = MusicLayout.Covers,
    val query: String = "",
    val groups: List<AudioGroup> = emptyList(),
    val selectedGroup: AudioGroup? = null,
    val selectedArtist: AudioGroup? = null,
    val artistAlbums: List<AudioGroup> = emptyList(),
    val tracks: List<AudioTrack> = emptyList(),
    val loading: Boolean = false,
    val loadingMore: Boolean = false,
    val nextCursor: String? = null,
    val nextOffset: Int? = null,
    val error: String? = null,
    val downloadedIds: Set<String> = emptySet(),
    val offlineTracks: List<AudioTrack> = emptyList(),
    val downloadedOnly: Boolean = false,
    val downloadingId: String? = null,
    val downloadMessage: String? = null,
    val favoriteIds: Set<String> = emptySet(),
    val favoriteTracks: List<AudioTrack> = emptyList(),
    val favoritesOnly: Boolean = false,
    val offlineOnly: Boolean = false,
    val discoveryAlbums: List<AudioGroup> = emptyList(),
    val discoveryArtists: List<AudioGroup> = emptyList(),
    val discoveryRecent: List<AudioTrack> = emptyList(),
    val discoveryPlayed: List<AudioTrack> = emptyList(),
    val batchDownloadCompleted: Int = 0,
    val batchDownloadTotal: Int = 0,
    val batchDownloadLabel: String = "",
    val actionMessage: String? = null,
    val navigation: List<AudioNavigationBucket> = emptyList(),
    val bucket: String = "",
)

data class AudioNavigationBucket(val bucket: String, val offset: Int, val count: Int)

internal fun MusicState.offlineFallback(): MusicState? {
    val available = offlineTracks.filter { libraryId.isBlank() || it.libraryId == libraryId }
    if (available.isEmpty()) return null
    return copy(
        grouping = MusicGrouping.Tracks,
        tracks = emptyList(),
        downloadedOnly = true,
        favoritesOnly = false,
        offlineOnly = true,
        loading = false,
        loadingMore = false,
        nextCursor = null,
        nextOffset = null,
        error = null,
    )
}

internal fun MusicState.withFavorite(track: AudioTrack, favorite: Boolean): MusicState {
    val key = track.identityKey
    return if (favorite) copy(
        favoriteIds = favoriteIds + key,
        favoriteTracks = if (favoriteTracks.any { it.identityKey == key }) favoriteTracks else favoriteTracks + track,
    ) else copy(
        favoriteIds = favoriteIds - key,
        favoriteTracks = favoriteTracks.filterNot { it.identityKey == key },
    )
}
