package net.kdhmedia.mochil.audio

/** The server's saved music queue is distinct from the active general audio queue. */
internal fun canPersistMusicPlaybackQueue(tracks: List<AudioTrack>): Boolean =
    tracks.isNotEmpty() && tracks.all { it.libraryKind == "music" }
