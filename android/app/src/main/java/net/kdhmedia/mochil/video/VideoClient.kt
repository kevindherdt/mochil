package net.kdhmedia.mochil.video

import java.net.HttpURLConnection
import java.net.URI
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.diagnostics.PerformanceDiagnostics
import net.kdhmedia.mochil.playback.PlaybackQuality
import net.kdhmedia.mochil.network.readBoundedResponseText
import net.kdhmedia.mochil.network.requireAtMost
import net.kdhmedia.mochil.network.openAndroidApiGet
import org.json.JSONArray
import org.json.JSONObject

class VideoClient {
    internal fun videos(credential: DeviceCredential, libraryId: String, show: VideoShow? = null, season: Int? = null, query: String = "", offset: Int = 0): VideoPage<VideoItem> =
        page(credential, "/api/android/v1/videos${videoQuery(libraryId, show, season, query, offset)}", ::video)

    internal fun shows(credential: DeviceCredential, libraryId: String, query: String = "", offset: Int = 0): VideoPage<VideoShow> =
        page(credential, "/api/android/v1/video-shows?library=${encode(libraryId)}&limit=$VIDEO_PAGE_SIZE&offset=${offset.coerceAtLeast(0)}${if (query.isBlank()) "" else "&q=${encode(query.trim())}"}") { value ->
            VideoShow(
                value.optString("name", "Untitled show"), value.optString("source_name"), value.optString("item_id"), value.optInt("count"),
                value.optJSONArray("seasons")?.requireAtMost(MAX_VIDEO_SEASONS, "Video seasons")?.mapObjects { it.optInt("number") }.orEmpty(), value.optString("artwork_url"),
                value.optInt("watched_count"), value.optInt("in_progress_count"), value.optInt("year"), value.optString("overview"), value.optString("backdrop_url"),
            )
        }

    /** The genres, runtime, age rating and score for one matched show (by its source name) or movie (by item id); empty when there is no match. */
    fun details(credential: DeviceCredential, libraryId: String, show: String?, itemId: String?): VideoTitleDetails {
        val path = "/api/android/v1/video-details/${encodePath(libraryId)}?" + if (show != null) "show=${encode(show)}" else "item=${encode(itemId.orEmpty())}"
        val connection = openCatalogue(credential, path)
        return try {
            if (connection.responseCode !in 200..299) return VideoTitleDetails()
            val value = JSONObject(connection.inputStream.use { it.readBoundedResponseText() })
            VideoTitleDetails(
                genres = value.optJSONArray("genres")?.let { list -> (0 until minOf(list.length(), 4)).map { list.optString(it) }.filter(String::isNotBlank) }.orEmpty(),
                runtimeMinutes = value.optInt("runtime_minutes"), rating = value.optDouble("rating", 0.0), certification = value.optString("certification"),
                cast = value.optJSONArray("cast")?.let { list ->
                    (0 until minOf(list.length(), 10)).mapNotNull { index ->
                        list.optJSONObject(index)?.let { member ->
                            member.optString("name", "").trim().takeIf(String::isNotEmpty)?.let { name ->
                                val portrait = member.optString("profile_path", "").trim().takeIf(String::isNotEmpty)
                                    ?.let { "/api/android/v1/tmdb-portrait?path=${encode(it)}" }.orEmpty()
                                VideoCastMember(name, member.optString("character", "").trim(), portrait)
                            }
                        }
                    }
                }.orEmpty(),
            )
        } finally { connection.disconnect() }
    }

    fun continueWatching(credential: DeviceCredential): List<VideoItem> =
        PerformanceDiagnostics.measure("network", "load_dashboard_watching") { array(credential, "/api/android/v1/video-progress?continue=1&limit=5").requireAtMost(5, "Continue watching") }.mapObjects(::video)

    fun dismissContinueWatching(credential: DeviceCredential, item: VideoItem) {
        patchProgress(credential, item, "dismissed", true)
    }

    fun setWatched(credential: DeviceCredential, item: VideoItem, watched: Boolean) {
        patchProgress(credential, item, "finished", watched)
    }

    private fun patchProgress(credential: DeviceCredential, item: VideoItem, field: String, value: Boolean) {
        val connection = (URI(credential.serverUrl + "/api/android/v1/video-progress/${encodePath(item.libraryId)}/${encodePath(item.id)}").toURL().openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"; connectTimeout = 10_000; readTimeout = 15_000; doOutput = true
            instanceFollowRedirects = false
            setRequestProperty("Authorization", "Bearer ${credential.token}"); setRequestProperty("Content-Type", "application/json")
        }
        try {
            connection.outputStream.use { it.write(JSONObject().put(field, value).toString().toByteArray()) }
            if (connection.responseCode !in 200..299) error("Video progress returned ${connection.responseCode}.")
        } finally { connection.disconnect() }
    }

    fun absolute(credential: DeviceCredential, path: String): String = if (path.isBlank()) "" else credential.serverUrl + when {
        path.startsWith("/api/video-artwork/") -> appendQuery(path.replaceFirst("/api/video-artwork/", "/api/android/v1/video-artwork/"), "client_artwork", "2")
        path.startsWith("/api/video-metadata-artwork/") -> path.replaceFirst("/api/video-metadata-artwork/", "/api/android/v1/video-metadata-artwork/")
        path.startsWith("/api/video-show-metadata-artwork/") -> path.replaceFirst("/api/video-show-metadata-artwork/", "/api/android/v1/video-show-metadata-artwork/")
        path.startsWith("/api/video-hls/") -> path.replaceFirst("/api/video-hls/", "/api/android/v1/video-hls/")
        path.startsWith("/videos/") -> path.replaceFirst("/videos/", "/api/android/v1/video-stream/")
        else -> path
    }

    fun playbackPath(path: String, quality: PlaybackQuality): String {
        if (quality == PlaybackQuality.Maximum || !path.contains("/video-hls/")) return path
        val marker = "/index.m3u8"
        return if (path.contains(marker)) path.replace(marker, "/${quality.wireValue}$marker") else path
    }

    fun subtitlePath(libraryId: String, itemId: String, index: Int) =
        "/api/android/v1/video-subtitles/${encodePath(libraryId)}/${encodePath(itemId)}/$index.vtt"

    internal fun video(value: JSONObject): VideoItem {
        val progress = value.optJSONObject("progress")
        return VideoItem(
            id = value.getString("id"), libraryId = value.getString("library_id"), title = value.optString("title", "Untitled video"), episodeTitle = value.optString("episode_title"),
            show = value.optString("show"), season = value.optInt("season"), episode = value.optInt("episode"),
            durationSeconds = value.optDouble("duration_seconds").takeIf(Double::isFinite) ?: 0.0,
            artworkPath = value.optString("artwork_url"), backdropPath = value.optString("backdrop_url"), revision = value.optString("revision"), streamPath = value.optString("stream_url"), hlsPath = value.optString("hls_url"),
            resumeSeconds = progress?.optDouble("position_seconds")?.takeIf(Double::isFinite) ?: 0.0,
            subtitles = value.optJSONArray("subtitle_streams")?.requireAtMost(MAX_VIDEO_SUBTITLES, "Video subtitles")?.let { subtitles ->
                (0 until subtitles.length()).mapNotNull { index -> subtitles.getJSONObject(index).takeIf { it.optBoolean("text") }?.let { subtitle ->
                    VideoSubtitle(subtitle.optInt("index"), subtitle.optString("language"), subtitle.optString("title"), subtitle.optBoolean("default"))
                } }
            }.orEmpty(),
            dismissed = progress?.optBoolean("dismissed") ?: false,
            finished = progress?.optBoolean("finished") ?: false,
            progressRevision = progress?.optLong("revision") ?: 0,
            year = value.optInt("year"), overview = value.optString("overview"),
        )
    }

    private fun array(credential: DeviceCredential, path: String): JSONArray {
        val connection = openCatalogue(credential, path)
        return try {
            if (connection.responseCode !in 200..299) error("Video catalogue returned ${connection.responseCode}.")
            JSONArray(connection.inputStream.use { it.readBoundedResponseText() })
        } finally { connection.disconnect() }
    }

    private fun <T> page(credential: DeviceCredential, path: String, transform: (JSONObject) -> T): VideoPage<T> {
        val connection = openCatalogue(credential, path)
        return try {
            if (connection.responseCode !in 200..299) error("Video catalogue returned ${connection.responseCode}.")
            val values = JSONArray(connection.inputStream.use { it.readBoundedResponseText() }).requireAtMost(VIDEO_PAGE_SIZE, "Video page")
            val items = values.mapObjects(transform)
            val total = connection.getHeaderField("X-Total-Count")?.toIntOrNull()?.coerceAtLeast(items.size) ?: items.size
            val offset = queryOffset(path)
            VideoPage(items, (offset + items.size).takeIf { it < total && items.isNotEmpty() })
        } finally { connection.disconnect() }
    }

    private fun openCatalogue(credential: DeviceCredential, path: String) =
        openAndroidApiGet(credential, path, readTimeoutMillis = 30_000)

    private fun encode(value: String) = URLEncoder.encode(value, StandardCharsets.UTF_8.toString())
    private fun encodePath(value: String) = encode(value).replace("+", "%20")
}

internal fun appendQuery(path: String, name: String, value: String): String =
    "$path${if ('?' in path) '&' else '?'}${URLEncoder.encode(name, StandardCharsets.UTF_8.toString())}=${URLEncoder.encode(value, StandardCharsets.UTF_8.toString())}"

internal const val VIDEO_PAGE_SIZE = 60
private const val MAX_VIDEO_SEASONS = 200
private const val MAX_VIDEO_SUBTITLES = 128
internal data class VideoPage<T>(val items: List<T>, val nextOffset: Int?)

internal fun videoQuery(libraryId: String, show: VideoShow?, season: Int?, search: String, offset: Int = 0): String = buildString {
    append("?library=").append(URLEncoder.encode(libraryId, StandardCharsets.UTF_8.toString())).append("&limit=").append(VIDEO_PAGE_SIZE).append("&offset=").append(offset.coerceAtLeast(0))
    if (show != null) {
        if (show.sourceName == "Other videos") append("&other=1")
        else append("&show=").append(URLEncoder.encode(show.sourceName, StandardCharsets.UTF_8.toString()))
    }
    if (season != null) append("&season=").append(season)
    if (search.isNotBlank()) append("&q=").append(URLEncoder.encode(search.trim(), StandardCharsets.UTF_8.toString()))
}

internal fun queryOffset(path: String): Int = Regex("(?:[?&])offset=(\\d+)").find(path)?.groupValues?.getOrNull(1)?.toIntOrNull() ?: 0

private fun <T> JSONArray.mapObjects(transform: (JSONObject) -> T): List<T> = List(length()) { transform(getJSONObject(it)) }
