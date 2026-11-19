package net.kdhmedia.mochil.audio

import java.net.HttpURLConnection
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.playback.PlaybackQuality
import org.json.JSONArray
import org.json.JSONObject

class AudioClient {
    fun navigation(credential: DeviceCredential, libraryId: String, grouping: MusicGrouping, search: String = ""): List<AudioNavigationBucket> {
        val mode = when (grouping) { MusicGrouping.Albums -> "albums"; MusicGrouping.Artists -> "artists"; else -> "tracks" }
        val query = buildString {
            append("?library=").append(encode(libraryId)).append("&mode=").append(mode).append("&sort=")
            append(if (grouping == MusicGrouping.Albums) "album" else if (grouping == MusicGrouping.Artists) "artist" else "title")
            if (search.isNotBlank()) append("&q=").append(encode(search.trim()))
        }
        val connection = openGet(credential, "/api/android/v1/audio-navigation$query")
        return try {
            val values = responseArray(connection).requireAtMost(MAX_NAVIGATION_BUCKETS, "Audio navigation")
            List(values.length()) { index -> values.getJSONObject(index).run { AudioNavigationBucket(getString("bucket"), optInt("offset"), optInt("count")) } }
        } finally { connection.disconnect() }
    }

    fun groups(credential: DeviceCredential, libraryId: String, grouping: MusicGrouping, offset: Int, search: String = "", limit: Int = 60): AudioGroupPage {
        val boundedLimit = limit.coerceIn(1, 120)
        val query = "?library=${encode(libraryId)}&group_type=${grouping.name.lowercase()}&limit=$boundedLimit&offset=$offset" +
            if (search.isBlank()) "" else "&q=${encode(search.trim())}"
        val connection = openGet(credential, "/api/android/v1/audio-groups$query")
        return try {
            val values = responseArray(connection).requireAtMost(boundedLimit, "Audio group page")
            AudioGroupPage(
                items = List(values.length()) { index -> values.getJSONObject(index).run {
                    AudioGroup(
                        name = getString("name"),
                        subtitle = optString("subtitle"),
                        count = optInt("count"),
                        artworkItemId = optString("artwork_item_id"),
                        artworkLibraryId = optString("artwork_library_id"),
                        id = optString("id"),
                    )
                } },
                nextOffset = connection.getHeaderField("X-Next-Offset")?.toIntOrNull(),
            )
        } finally { connection.disconnect() }
    }

    fun groupTracks(credential: DeviceCredential, libraryId: String, grouping: MusicGrouping, group: String, artist: String? = null, groupId: String = ""): List<AudioTrack> {
        val query = "?library=${encode(libraryId)}&group_type=${grouping.name.lowercase()}&group=${encode(group)}" + artist?.let { "&artist=${encode(it)}" }.orEmpty() + groupId.takeIf(String::isNotBlank)?.let { "&group_id=${encode(it)}" }.orEmpty()
        val connection = openGet(credential, "/api/android/v1/audio-group-tracks$query")
        return try { parseTracks(responseArray(connection).requireAtMost(MAX_AUDIO_ITEMS, "Audio group")) } finally { connection.disconnect() }
    }

    fun artistAlbums(credential: DeviceCredential, libraryId: String, artist: String): List<AudioGroup> {
        val query = "?library=${encode(libraryId)}&artist=${encode(artist)}"
        val connection = openGet(credential, "/api/android/v1/audio-artist-albums$query")
        return try {
            val values = responseArray(connection).requireAtMost(MAX_AUDIO_GROUPS, "Artist albums")
            List(values.length()) { index -> values.getJSONObject(index).run {
                AudioGroup(getString("name"), optString("subtitle"), optInt("count"), optString("artwork_item_id"), optString("artwork_library_id"), optString("id"))
            } }
        } finally { connection.disconnect() }
    }

    fun tracks(
        credential: DeviceCredential,
        libraryId: String,
        cursor: String?,
        search: String = "",
        limit: Int = 80,
        sort: String = "",
        played: String = "",
        offset: Int = 0,
    ): AudioPage {
        val query = buildString {
            append("?library=").append(encode(libraryId)).append("&limit=").append(limit.coerceIn(1, 500))
            if (search.isNotBlank()) append("&q=").append(encode(search.trim()))
            if (sort.isNotBlank()) append("&sort=").append(encode(sort))
            if (played.isNotBlank()) append("&played=").append(encode(played))
            if (!cursor.isNullOrBlank()) append("&cursor=").append(encode(cursor))
            else if (offset > 0) append("&offset=").append(offset)
        }
        val connection = openGet(credential, "/api/android/v1/audio$query")
        return try {
            val values = responseArray(connection).requireAtMost(limit.coerceIn(1, 500), "Audio page")
            AudioPage(
                items = parseTracks(values),
                nextCursor = connection.getHeaderField("X-Next-Cursor")?.takeIf(String::isNotBlank),
            )
        } finally {
            connection.disconnect()
        }
    }

    fun favorites(credential: DeviceCredential): List<AudioTrack> {
        val connection = openGet(credential, "/api/android/v1/audio-favorites")
        return try { parseTracks(responseArray(connection).requireAtMost(MAX_AUDIO_ITEMS, "Audio favorites")) } finally { connection.disconnect() }
    }

    fun recentlyPlayed(credential: DeviceCredential, libraryId: String, limit: Int = 8): List<AudioTrack> {
        val query = "?music_library=${encode(libraryId)}&limit=${limit.coerceIn(1, 20)}"
        val connection = openGet(credential, "/api/android/v1/audio-progress$query")
        return try { parseTracks(responseArray(connection).requireAtMost(limit.coerceIn(1, 20), "Recently played audio")) } finally { connection.disconnect() }
    }

    fun setFavorite(credential: DeviceCredential, track: AudioTrack, favorite: Boolean) {
        val connection = openMutation(credential, "/api/android/v1/audio-favorites").apply {
            requestMethod = "PUT"
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        val body = JSONObject().put("library_id", track.libraryId).put("item_id", track.id).put("favorite", favorite).toString()
        try {
            connection.outputStream.use { it.write(body.toByteArray()) }
            if (connection.responseCode !in 200..299) throw AudioClientException(connection.responseCode)
        } finally { connection.disconnect() }
    }

    /** A file already at or near the target bitrate gains nothing from being re-encoded; an unknown bitrate is converted. */
    private fun worthConverting(bitrate: Long, quality: PlaybackQuality): Boolean {
        val target = when (quality) { PlaybackQuality.High -> 192_000L; PlaybackQuality.Medium -> 128_000L; PlaybackQuality.Low -> 80_000L; else -> return false }
        return bitrate < 1_000 || bitrate > target * 11 / 10
    }

    fun streamUrl(credential: DeviceCredential, track: AudioTrack) =
        "${credential.serverUrl}/api/android/v1/audio-stream/${path(track.libraryId)}/${path(track.id)}/audio"

    fun playbackUrl(credential: DeviceCredential, track: AudioTrack, quality: PlaybackQuality): String =
        if (quality == PlaybackQuality.Maximum || !worthConverting(track.bitrate, quality)) streamUrl(credential, track)
        else "${credential.serverUrl}/api/android/v1/audio-hls/${path(track.libraryId)}/${path(track.id)}/${quality.wireValue}/index.m3u8"

    fun artworkUrl(credential: DeviceCredential, track: AudioTrack) =
        "${credential.serverUrl}/api/android/v1/audio-artwork/${path(track.libraryId)}/${path(track.id)}?size=card"

    fun artworkUrl(credential: DeviceCredential, group: AudioGroup) =
        "${credential.serverUrl}/api/android/v1/audio-artwork/${path(group.artworkLibraryId)}/${path(group.artworkItemId)}?size=card"

    private fun openGet(credential: DeviceCredential, path: String) = openAndroidApiGet(credential, path)

    private fun openMutation(credential: DeviceCredential, path: String) =
        (java.net.URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 10_000
            readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun responseArray(connection: HttpURLConnection): JSONArray {
        if (connection.responseCode !in 200..299) throw AudioClientException(connection.responseCode)
        return JSONArray(connection.inputStream.use { it.readBoundedResponseText() })
    }

    private fun parseTracks(values: JSONArray) = List(values.length()) { index -> parseAudioTrack(values.getJSONObject(index)) }

    private fun parseAudioTrack(value: JSONObject) = value.run {
        val progress = optJSONObject("progress")
        AudioTrack(
            id = getString("id"), libraryId = getString("library_id"),
            title = optString("title", "Untitled track"), artist = optString("artist", "Unknown artist"),
            album = optString("album"), durationSeconds = optDouble("duration_seconds"), mediaType = optString("media_type", "audio/mpeg"),
            series = optString("series"), seriesIndex = optDouble("series_index"), narrator = optString("narrator"),
            libraryKind = optString("library_kind"),
            sizeBytes = optLong("size").coerceAtLeast(0),
            albumArtist = optString("album_artist"), genre = optString("genre"),
            trackNumber = optInt("track").coerceAtLeast(0), discNumber = optInt("disc").coerceAtLeast(0),
            container = optString("container"),
            year = optInt("year").takeIf { it in 1000..2100 } ?: 0,
            bitrate = optLong("bitrate").takeIf { it in 1_000..1_000_000_000 } ?: 0,
            codec = optString("codec"), sampleRate = optInt("sample_rate").coerceAtLeast(0), bitDepth = optInt("bit_depth").coerceAtLeast(0),
            resumePositionSeconds = progress?.optDouble("position_seconds")?.takeIf(Double::isFinite) ?: 0.0,
            finished = progress?.optBoolean("finished") ?: false,
            replayTrackDb = optionalFiniteDouble("replaygain_track_db"), replayAlbumDb = optionalFiniteDouble("replaygain_album_db"),
            replayTrackPeak = optionalFiniteDouble("replaygain_track_peak"), replayAlbumPeak = optionalFiniteDouble("replaygain_album_peak"),
            chapters = optJSONArray("chapters")?.requireAtMost(MAX_AUDIO_CHAPTERS, "Audio chapters")?.let { chapters -> List(chapters.length()) { chapterIndex -> chapters.getJSONObject(chapterIndex).run {
                AudioChapter(optString("title", "Chapter ${chapterIndex + 1}"), optDouble("start_seconds").finiteOrZero(), optDouble("end_seconds").finiteOrZero())
            } } }.orEmpty(),
        )
    }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    private fun path(value: String) = encode(value).replace("+", "%20")
}

private fun Double.finiteOrZero() = takeIf(Double::isFinite) ?: 0.0

class AudioClientException(val status: Int) : Exception("The server returned $status.")

private const val MAX_NAVIGATION_BUCKETS = 128
private const val MAX_AUDIO_GROUPS = 2_000
private const val MAX_AUDIO_ITEMS = 10_000
private const val MAX_AUDIO_CHAPTERS = 2_000
