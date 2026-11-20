package net.kdhmedia.mochil.audio

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONArray
import org.json.JSONObject

data class MusicPlaylist(val name: String, val tracks: List<AudioTrack>, val revision: String? = null, val visibleEdit: Boolean = false)

internal class MusicPlaylistConflict : IllegalStateException("Playlist changed on another device. Review it before trying again.")
internal class MusicPlaylistMissing : IllegalStateException("Playlist no longer exists")

class MusicPlaylistClient internal constructor(private val connectionFactory: (URI) -> HttpURLConnection) {
    constructor() : this({ it.toURL().openConnection() as HttpURLConnection })
    fun overview(credential: DeviceCredential): List<MusicPlaylistOverview> {
        val connection = open(credential, "/api/android/v1/audio-playlists?view=summary", "GET")
        return try {
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
            val text = connection.inputStream.use { it.readBoundedResponseText() }
            if (text.trimStart().startsWith("[")) {
                // Older servers ignore view=summary and return full arrays.
                val lists = JSONArray(text).requireAtMost(MAX_PLAYLISTS, "Music playlists")
                List(lists.length()) { index ->
                    val list = lists.getJSONObject(index)
                    val items = (list.optJSONArray("items") ?: JSONArray()).requireAtMost(MAX_PLAYLIST_ITEMS, "Music playlist")
                    val artwork = linkedSetOf<String>()
                    for (i in 0 until items.length()) {
                        items.getJSONObject(i).optString("artwork_url").takeIf { it.isNotBlank() }?.let(artwork::add)
                        if (artwork.size == 4) break
                    }
                    MusicPlaylistOverview(list.getString("name"), items.length(), artwork.toList())
                }
            } else parsePlaylistSummaries(JSONObject(text))
        } finally { connection.disconnect() }
    }

    fun get(credential: DeviceCredential, name: String): MusicPlaylist {
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(name)}", "GET")
        try {
            when (val status = connection.responseCode.also { net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(it) }) {
                HttpURLConnection.HTTP_BAD_METHOD -> Unit // Older servers only offered collection GET.
                HttpURLConnection.HTTP_NOT_FOUND -> throw MusicPlaylistMissing()
                in 200..299 -> {
                    val value = JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
                    val actualName = value.getString("name")
                    check(actualName.equals(name, ignoreCase = true)) { "The server returned a different playlist." }
                    // A summary or malformed response must never become an empty replacement.
                    val items = value.getJSONArray("items").requireAtMost(MAX_PLAYLIST_ITEMS, "Music playlist")
                    return MusicPlaylist(actualName, parseTracks(items), connection.getHeaderField("ETag"),
                        connection.getHeaderField("X-Mochil-Playlist-Edit") == "visible-v1")
                }
                else -> error("The server returned $status.")
            }
        } finally { connection.disconnect() }
        return list(credential).firstOrNull { it.name.equals(name, ignoreCase = true) }
            ?: throw MusicPlaylistMissing()
    }

    fun list(credential: DeviceCredential): List<MusicPlaylist> {
        val connection = open(credential, "/api/android/v1/audio-playlists", "GET")
        return try {
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
            val values = JSONArray(connection.inputStream.use { it.readBoundedResponseText() }).requireAtMost(MAX_PLAYLISTS, "Music playlists")
            List(values.length()) { index -> values.getJSONObject(index).run {
                MusicPlaylist(getString("name"), parseTracks((optJSONArray("items") ?: JSONArray()).requireAtMost(MAX_PLAYLIST_ITEMS, "Music playlist")))
            } }
        } finally { connection.disconnect() }
    }

    fun create(credential: DeviceCredential, name: String) {
        val connection = open(credential, "/api/android/v1/audio-playlists", "POST").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        request(connection, JSONObject().put("name", name).toString())
    }

    fun put(credential: DeviceCredential, name: String, tracks: List<AudioTrack>, expectedRevision: String? = null) {
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(name)}", "PUT").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            if (!expectedRevision.isNullOrBlank()) setRequestProperty("If-Match", expectedRevision)
        }
        val body = JSONArray().apply {
            tracks.forEach { put(JSONObject().put("library_id", it.libraryId).put("item_id", it.id)) }
        }
        request(connection, body.toString())
    }

    fun add(credential: DeviceCredential, name: String, track: AudioTrack) {
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(name)}", "POST").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        request(connection, JSONObject().put("library_id", track.libraryId).put("item_id", track.id).toString())
    }

    fun editVisible(credential: DeviceCredential, original: MusicPlaylist, tracks: List<AudioTrack>) {
        if (!original.visibleEdit) return put(credential, original.name, tracks, original.revision)
        check(!original.revision.isNullOrBlank()) { "Could not verify this playlist revision. No changes were sent." }
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(original.name)}?edit=visible", "PUT").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setRequestProperty("If-Match", original.revision)
        }
        fun entries(values: List<AudioTrack>) = JSONArray().apply {
            values.forEach { put(JSONObject().put("library_id", it.libraryId).put("item_id", it.id)) }
        }
        request(connection, JSONObject().put("visible_entries", entries(original.tracks)).put("entries", entries(tracks)).toString())
    }

    fun delete(credential: DeviceCredential, name: String, expectedRevision: String? = null) {
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(name)}", "DELETE").apply {
            if (!expectedRevision.isNullOrBlank()) setRequestProperty("If-Match", expectedRevision)
        }
        request(connection)
    }

    fun rename(credential: DeviceCredential, name: String, newName: String, expectedRevision: String? = null) {
        val connection = open(credential, "/api/android/v1/audio-playlists/${segment(name)}", "PATCH").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
            if (!expectedRevision.isNullOrBlank()) setRequestProperty("If-Match", expectedRevision)
        }
        request(connection, JSONObject().put("name", newName).toString())
    }

    private fun request(connection: HttpURLConnection, body: String? = null) {
        try {
            if (body != null) connection.outputStream.use { it.write(body.toByteArray()) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status == HttpURLConnection.HTTP_PRECON_FAILED) throw MusicPlaylistConflict()
            if (status !in 200..299) error("The server returned $status.")
        } finally { connection.disconnect() }
    }

    private fun open(credential: DeviceCredential, path: String, method: String) =
        connectionFactory(URI(credential.serverUrl + path)).apply {
            requestMethod = method
            connectTimeout = 10_000
            readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun parseTracks(values: JSONArray) = List(values.length()) { index -> values.getJSONObject(index).run {
        AudioTrack(
            getString("id"), getString("library_id"), optString("title", "Untitled track"), optString("artist", "Unknown artist"), optString("album"),
            optDouble("duration_seconds"), optString("media_type", "audio/mpeg"), optionalFiniteDouble("replaygain_track_db"), optionalFiniteDouble("replaygain_album_db"),
            optionalFiniteDouble("replaygain_track_peak"), optionalFiniteDouble("replaygain_album_peak"),
            libraryKind = optString("library_kind"), year = optInt("year").takeIf { it in 1000..2100 } ?: 0,
            bitrate = optLong("bitrate").takeIf { it in 1_000..1_000_000_000 } ?: 0,
            codec = optString("codec"), sampleRate = optInt("sample_rate").coerceAtLeast(0), bitDepth = optInt("bit_depth").coerceAtLeast(0),
        )
    } }

    private fun segment(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString()).replace("+", "%20")
}

private const val MAX_PLAYLISTS = 1_000
private const val MAX_PLAYLIST_ITEMS = 10_000
