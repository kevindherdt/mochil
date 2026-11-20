package net.kdhmedia.mochil.audio

/** Leaving detail restores cached parent content without retaining its pending spinner. */
internal fun MusicState.withClosedMusicGroup(): MusicState {
    if (selectedGroup == null) return this
    return copy(selectedGroup = null, tracks = emptyList(), loading = false,
        loadingMore = false, error = null)
}

internal fun MusicState.withClosedMusicArtist(): MusicState {
    if (selectedArtist == null) return this
    return copy(selectedArtist = null, selectedGroup = null, artistAlbums = emptyList(),
        tracks = emptyList(), loading = false, loadingMore = false, error = null)
}
