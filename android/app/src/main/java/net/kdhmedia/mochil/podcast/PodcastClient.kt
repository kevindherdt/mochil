package net.kdhmedia.mochil.podcast

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.network.MAX_ERROR_RESPONSE_BYTES
import net.kdhmedia.mochil.network.openAndroidApiGet
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import org.json.JSONArray
import org.json.JSONObject

class PodcastClient {
    fun discover(credential: DeviceCredential, query: String): List<PodcastDiscoveryResult> {
        val connection = openGet(credential, "/api/android/v1/podcast-search?q=${encode(query.trim())}")
        return try {
            val values = responseArray(connection).requireAtMost(MAX_DISCOVERY_RESULTS, "Podcast discovery")
            List(values.length()) { index -> values.getJSONObject(index).run {
                PodcastDiscoveryResult(
                    title = optString("title", "Untitled podcast"), author = optString("author"),
                    artworkPath = optString("artwork_url"), feedUrl = getString("feed_url"),
                    episodeCount = optInt("episode_count"), genre = optString("genre"),
                )
            } }
        } finally { connection.disconnect() }
    }

    fun subscribe(credential: DeviceCredential, feedUrl: String) {
        val body = JSONObject().put("url", feedUrl.trim()).toString().encodeToByteArray()
        val connection = openMutation(credential, "/api/android/v1/podcast-feeds").apply {
            requestMethod = "POST"; doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setFixedLengthStreamingMode(body.size)
        }
        try {
            connection.outputStream.use { it.write(body) }
            ensureSuccess(connection)
        } finally { connection.disconnect() }
    }

    fun unsubscribe(credential: DeviceCredential, showId: String) {
        val connection = openMutation(credential, "/api/android/v1/podcast-feeds/${segment(showId)}").apply { requestMethod = "DELETE" }
        try { ensureSuccess(connection) } finally { connection.disconnect() }
    }

    fun refresh(credential: DeviceCredential, showId: String) {
        val connection = openMutation(credential, "/api/android/v1/podcast-feeds/${segment(showId)}/refresh").apply {
            requestMethod = "POST"; setFixedLengthStreamingMode(0)
        }
        try { ensureSuccess(connection) } finally { connection.disconnect() }
    }

    fun updatePolicy(credential: DeviceCredential, showId: String, policy: PodcastPolicy) {
        val body = JSONObject().apply {
            put("auto_download", policy.autoDownload)
            put("retain_downloads", policy.retainDownloads)
            put("delete_played_after_days", policy.deletePlayedAfterDays)
        }.toString().encodeToByteArray()
        val connection = openMutation(credential, "/api/android/v1/podcast-feeds/${segment(showId)}/policy").apply {
            requestMethod = "PUT"; doOutput = true
            setRequestProperty("Content-Type", "application/json")
            setFixedLengthStreamingMode(body.size)
        }
        try { connection.outputStream.use { it.write(body) }; ensureSuccess(connection) } finally { connection.disconnect() }
    }

    fun shows(credential: DeviceCredential, offset: Int, query: String): PodcastShowPage {
        val path = buildString {
            append("/api/android/v1/podcast-shows?limit=60&offset=").append(offset)
            if (query.isNotBlank()) append("&q=").append(encode(query.trim()))
        }
        val connection = openGet(credential, path)
        return try {
            val values = responseArray(connection).requireAtMost(MAX_SHOW_PAGE_SIZE, "Podcast show page")
            PodcastShowPage(
                List(values.length()) { index -> values.getJSONObject(index).run {
                    val policy = optJSONObject("policy")
                    PodcastShow(
                        getString("id"), optString("title", "Untitled show"), optString("description"), optInt("episode_count"),
                        PodcastPolicy(policy?.optBoolean("auto_download") ?: false, policy?.optInt("retain_downloads") ?: 0, policy?.optInt("delete_played_after_days") ?: 0),
                        optString("last_error"),
                    )
                } },
                connection.getHeaderField("X-Next-Offset")?.toIntOrNull(),
            )
        } finally { connection.disconnect() }
    }

    fun episodes(credential: DeviceCredential, libraryId: String, show: PodcastShow, filter: PodcastPlayedFilter, sort: PodcastEpisodeSort, cursor: String?): PodcastEpisodePage {
        val path = buildString {
            append("/api/android/v1/audio?library=").append(encode(libraryId))
            append("&feed=").append(encode(show.id)).append("&sort=").append(sort.queryValue).append("&limit=80")
            if (filter != PodcastPlayedFilter.All) append("&played=").append(filter.name.lowercase())
            if (!cursor.isNullOrBlank()) append("&cursor=").append(encode(cursor))
        }
        val connection = openGet(credential, path)
        return try {
            val values = responseArray(connection).requireAtMost(MAX_EPISODE_PAGE_SIZE, "Podcast episode page")
            PodcastEpisodePage(List(values.length()) { index -> values.getJSONObject(index).run {
                val progress = optJSONObject("progress")
                AudioTrack(
                    id = getString("id"), libraryId = getString("library_id"), title = optString("title", "Untitled episode"),
                    artist = optString("artist", show.title), album = optString("album", show.title), durationSeconds = optDouble("duration_seconds"),
                    mediaType = optString("media_type", "audio/mpeg"), libraryKind = "podcasts",
                    resumePositionSeconds = progress?.optDouble("position_seconds")?.takeIf(Double::isFinite) ?: 0.0,
                    finished = progress?.optBoolean("finished") ?: false,
                    feedId = optString("feed_id", show.id),
                    description = optString("description").take(8_001),
                    publishedAtMillis = runCatching { java.time.Instant.parse(optString("modified")).toEpochMilli() }.getOrDefault(0),
                    sizeBytes = optLong("size").coerceAtLeast(0),
                    streamPath = optString("stream_url"), artworkPath = optString("artwork_url"),
                    serverDownloaded = optBoolean("server_downloaded"),
                )
            } }, connection.getHeaderField("X-Next-Cursor")?.takeIf(String::isNotBlank))
        } finally { connection.disconnect() }
    }

    /**
     * The audio list names a web route (`/podcast-media/…`) that does not accept a device token, so the player got an
     * error page instead of audio. The device route serves both streamed and server-downloaded episodes.
     */
    fun streamUrl(credential: DeviceCredential, episode: AudioTrack) =
        "${credential.serverUrl}/api/android/v1/podcast-stream/${segment(episode.id)}"

    fun latestProgress(credential: DeviceCredential, episode: AudioTrack): AudioTrack {
        val path = "/api/android/v1/audio-progress/${segment(episode.libraryId)}/${segment(episode.id)}"
        // Playback already has the position from the episode page. This refresh
        // improves cross-device accuracy, but must never hold the Play action
        // behind the longer catalogue timeout on a degraded connection.
        val connection = openAndroidApiGet(credential, path, connectTimeoutMillis = 5_000, readTimeoutMillis = 6_000)
        return try {
            when (connection.responseCode) {
                in 200..299 -> JSONObject(connection.inputStream.use { it.readBoundedResponseText() }).run {
                    episode.withPodcastProgress(
                        positionSeconds = optDouble("position_seconds").takeIf(Double::isFinite) ?: 0.0,
                        finished = optBoolean("finished"),
                    )
                }
                404 -> episode.withPodcastProgress(0.0, false)
                else -> throw PodcastClientException(connection.responseCode, "Episode progress could not be refreshed.")
            }
        } finally { connection.disconnect() }
    }

    fun setPlayed(credential: DeviceCredential, episode: AudioTrack, played: Boolean) {
        val path = "/api/android/v1/audio-progress/${segment(episode.libraryId)}/${segment(episode.id)}"
        val connection = openMutation(credential, path).apply {
            requestMethod = "POST"
            doOutput = true
            setRequestProperty("Content-Type", "application/json")
        }
        try {
            connection.outputStream.use { it.write(JSONObject().put("finished", played).toString().toByteArray()) }
            val status = connection.responseCode
            net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
            if (status !in 200..299) throw PodcastClientException(status, "Episode status could not be updated.")
        } finally { connection.disconnect() }
    }

    /** The audio list names the web route (`/api/podcast-artwork/…`), which rejects a device token; only a device route is used as listed. */
    fun episodeArtworkUrl(credential: DeviceCredential, episode: AudioTrack) =
        episode.artworkPath.takeIf { it.startsWith("/api/android/v1/") }?.let { credential.serverUrl + it }
            ?: "${credential.serverUrl}/api/android/v1/podcast-artwork/${segment(episode.id)}"

    fun showArtworkUrl(credential: DeviceCredential, show: PodcastShow) =
        "${credential.serverUrl}/api/android/v1/podcast-artwork/feed/${segment(show.id)}"

    fun discoveryArtworkUrl(credential: DeviceCredential, result: PodcastDiscoveryResult) =
        result.artworkPath.takeIf { it.startsWith("/api/android/v1/podcast-search-artwork?") }?.let { credential.serverUrl + it }.orEmpty()

    private fun openGet(credential: DeviceCredential, path: String) = openAndroidApiGet(credential, path)

    private fun openMutation(credential: DeviceCredential, path: String) =
        (URI(credential.serverUrl + path).toURL().openConnection() as HttpURLConnection).apply {
            connectTimeout = 10_000; readTimeout = 20_000
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}")
            setRequestProperty("Accept", "application/json")
        }

    private fun responseArray(connection: HttpURLConnection): JSONArray {
        ensureSuccess(connection)
        return JSONArray(connection.inputStream.use { it.readBoundedResponseText() })
    }

    private fun ensureSuccess(connection: HttpURLConnection) {
        val status = connection.responseCode
        net.kdhmedia.mochil.network.reportIfExpiredDeviceSession(status)
        if (status !in 200..299) {
            val detail = runCatching { connection.errorStream?.use { it.readBoundedResponseText(MAX_ERROR_RESPONSE_BYTES) }?.trim() }.getOrNull().orEmpty()
            throw PodcastClientException(status, detail)
        }
    }

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    private fun segment(value: String) = encode(value).replace("+", "%20")
}

private const val MAX_DISCOVERY_RESULTS = 20
private const val MAX_SHOW_PAGE_SIZE = 60
private const val MAX_EPISODE_PAGE_SIZE = 80

internal class PodcastClientException(val status: Int, detail: String = "") : Exception(detail.ifBlank { "The server returned $status." })

internal fun AudioTrack.withPodcastProgress(positionSeconds: Double, finished: Boolean): AudioTrack {
    val duration = durationSeconds.takeIf { it.isFinite() && it > 0.0 }
    val bounded = positionSeconds.takeIf(Double::isFinite)?.coerceAtLeast(0.0)?.let { position ->
        duration?.let { position.coerceAtMost(it) } ?: position
    } ?: 0.0
    return copy(resumePositionSeconds = if (finished && duration != null) duration else bounded, finished = finished)
}
