package net.kdhmedia.mochil.audio

/** Root cursors belong only to the online root view, never to a detail or local filter. */
internal fun MusicState.canLoadMoreMusic(): Boolean =
    !loading && !loadingMore && selectedGroup == null && selectedArtist == null &&
        !downloadedOnly && !favoritesOnly && !offlineOnly &&
        if (grouping.isPagedTrackView()) nextCursor != null else nextOffset != null
