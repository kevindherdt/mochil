package net.kdhmedia.mochil.audio

import java.net.HttpURLConnection
import java.net.URI
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONArray
import org.json.JSONObject

class MusicQueueClient {
    fun get(credential: DeviceCredential): List<AudioTrack> {
        val connection = open(credential, "GET")
        return try {
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
            parseTracks(JSONArray(connection.inputStream.use { it.readBoundedResponseText() }).requireAtMost(MAX_QUEUE_ITEMS, "Music queue"))
        } finally { connection.disconnect() }
    }

    fun put(credential: DeviceCredential, tracks: List<AudioTrack>) {
        val connection = open(credential, "PUT").apply {
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        val body = JSONArray().apply { tracks.forEach { put(JSONObject().put("library_id", it.libraryId).put("item_id", it.id)) } }.toString()
        try {
            connection.outputStream.use { it.write(body.toByteArray()) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) error("The server returned $status.")
        } finally { connection.disconnect() }
    }

    private fun open(credential: DeviceCredential, method: String) =
        (URI(credential.serverUrl + "/api/android/v1/audio-queue").toURL().openConnection() as HttpURLConnection).apply {
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
}

private const val MAX_QUEUE_ITEMS = 10_000
