package net.kdhmedia.mochil.audio

data class AudioTrack(
    val id: String,
    val libraryId: String,
    val title: String,
    val artist: String,
    val album: String,
    val durationSeconds: Double,
    val mediaType: String = "audio/mpeg",
    val replayTrackDb: Double? = null,
    val replayAlbumDb: Double? = null,
    val replayTrackPeak: Double? = null,
    val replayAlbumPeak: Double? = null,
    val series: String = "",
    val seriesIndex: Double = 0.0,
    val narrator: String = "",
    val libraryKind: String = "",
    val resumePositionSeconds: Double = 0.0,
    val finished: Boolean = false,
    val feedId: String = "",
    val description: String = "",
    val publishedAtMillis: Long = 0,
    val sizeBytes: Long = 0,
    val chapters: List<AudioChapter> = emptyList(),
    val albumArtist: String = "",
    val genre: String = "",
    val trackNumber: Int = 0,
    val discNumber: Int = 0,
    val container: String = "",
    val year: Int = 0,
    val bitrate: Long = 0,
    val codec: String = "",
    val sampleRate: Int = 0,
    val bitDepth: Int = 0,
    val playbackMode: String = "",
    val playbackQuality: String = "",
    val streamPath: String = "",
    val artworkPath: String = "",
    val serverDownloaded: Boolean = false,
)

/** Stable across every client store even when either server ID contains punctuation. */
internal val AudioTrack.identityKey: String get() = audioTrackIdentityKey(libraryId, id)

internal fun audioTrackIdentityKey(libraryId: String, itemId: String): String =
    "${libraryId.length}:$libraryId${itemId.length}:$itemId"

data class AudioChapter(val title: String, val startSeconds: Double, val endSeconds: Double)

internal fun org.json.JSONObject.optionalFiniteDouble(name: String): Double? =
    if (has(name) && !isNull(name)) optDouble(name).takeIf(Double::isFinite) else null

data class AudioPage(val items: List<AudioTrack>, val nextCursor: String?)

enum class MusicGrouping { Discover, Continue, Recent, RecentlyPlayed, Albums, Artists, Favorites, Playlists, Series, Authors, Tracks }
enum class MusicLayout { Covers, List }

internal fun MusicGrouping.isPagedTrackView(): Boolean =
    this == MusicGrouping.Continue || this == MusicGrouping.Recent || this == MusicGrouping.Tracks

data class AudioGroup(
    val name: String,
    val subtitle: String,
    val count: Int,
    val artworkItemId: String,
    val artworkLibraryId: String,
    val id: String = "",
)

/** Stable UI identity for groups whose display names are not necessarily unique. */
fun AudioGroup.stableKey(): String = id.ifBlank {
    listOf(name, subtitle, artworkLibraryId, artworkItemId).joinToString("\u001f")
}

fun AudioGroup.hasSameIdentity(other: AudioGroup?): Boolean =
    other != null && stableKey() == other.stableKey()

data class AudioGroupPage(val items: List<AudioGroup>, val nextOffset: Int?)

fun MusicState.visibleTracks(): List<AudioTrack> {
    if (grouping == MusicGrouping.RecentlyPlayed) return discoveryPlayed.filter { libraryId.isBlank() || it.libraryId == libraryId }
    if (!downloadedOnly && !favoritesOnly && grouping != MusicGrouping.Favorites) return tracks
    val available = (if (downloadedOnly) offlineTracks else favoriteTracks).filter { libraryId.isBlank() || it.libraryId == libraryId }
    val group = selectedGroup ?: return available
    return when {
        selectedArtist != null -> available.filter { it.album.equals(group.name, ignoreCase = true) && it.artist.equals(selectedArtist.name, ignoreCase = true) }
        grouping == MusicGrouping.Albums -> available.filter { it.album.equals(group.name, ignoreCase = true) }
        grouping == MusicGrouping.Artists || grouping == MusicGrouping.Authors -> available.filter { it.artist.equals(group.name, ignoreCase = true) }
        grouping == MusicGrouping.Series -> available.filter { (it.series.ifBlank { it.album }).equals(group.name, ignoreCase = true) }
        else -> available
    }
}
