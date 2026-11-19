@file:android.annotation.SuppressLint("ApplySharedPref") // Lifecycle shutdown snapshots explicitly request synchronous durability.
package net.kdhmedia.mochil.playback

import android.content.Context
import android.net.Uri
import android.os.Bundle
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.Player
import java.net.URLEncoder
import java.net.URI
import java.io.File
import net.kdhmedia.mochil.audio.AudioChapter
import net.kdhmedia.mochil.audio.AudioTrack
import net.kdhmedia.mochil.auth.DeviceCredential
import net.kdhmedia.mochil.webdav.WEB_DAV_STREAM_SCHEME
import net.kdhmedia.mochil.webdav.parseWebDavStreamToken
import org.json.JSONArray
import org.json.JSONObject

internal data class PlaybackSnapshot(
    val items: List<MediaItem>,
    val startIndex: Int,
    val positionMs: Long,
    val speed: Float,
    val repeatMode: Int,
    val shuffle: Boolean,
)

internal enum class PlaybackSnapshotAction { Save, Clear }

internal fun playbackSnapshotAction(mediaItemCount: Int): PlaybackSnapshotAction =
    if (mediaItemCount > 0) PlaybackSnapshotAction.Save else PlaybackSnapshotAction.Clear

internal fun AudioTrack.toPlaybackMediaItem(
    uri: String,
    automaticStartPositionMs: Long? = null,
    automaticSeekToken: String = "",
): MediaItem = MediaItem.Builder()
    .setMediaId(id)
    .setUri(uri)
    .setMediaMetadata(MediaMetadata.Builder()
        .setTitle(title).setArtist(artist).setAlbumTitle(album)
        .setExtras(Bundle().apply {
            putString("library_id", libraryId); putString("library_kind", libraryKind)
            val source = audioPlaybackSource(uri)
            putString("playback_mode", source.mode); putString("playback_quality", source.quality)
            if (durationSeconds.isFinite() && durationSeconds > 0.0) putDouble("duration_seconds", durationSeconds)
            automaticStartPositionMs?.takeIf { it > 0L }?.let {
                putLong("automatic_start_position_ms", it)
                putString("automatic_seek_token", automaticSeekToken.take(80))
            }
            if (chapters.isNotEmpty()) putString("chapters", encodePlaybackChapters(chapters))
            replayTrackDb?.let { putDouble("replay_track_db", it) }; replayAlbumDb?.let { putDouble("replay_album_db", it) }
            replayTrackPeak?.let { putDouble("replay_track_peak", it) }; replayAlbumPeak?.let { putDouble("replay_album_peak", it) }
        })
        .setIsBrowsable(false).setIsPlayable(true).build())
    .build()

internal fun MediaItem.toPlaybackTrack(durationMs: Long = 0): AudioTrack {
    val extras = mediaMetadata.extras
    return AudioTrack(
        id = mediaId, libraryId = extras?.getString("library_id").orEmpty(),
        title = mediaMetadata.title?.toString().orEmpty(), artist = mediaMetadata.artist?.toString().orEmpty(),
        album = mediaMetadata.albumTitle?.toString().orEmpty(),
        durationSeconds = extras?.optionalPlaybackDouble("duration_seconds")
            ?: durationMs.coerceAtLeast(0) / 1_000.0,
        replayTrackDb = extras?.optionalPlaybackDouble("replay_track_db"), replayAlbumDb = extras?.optionalPlaybackDouble("replay_album_db"),
        replayTrackPeak = extras?.optionalPlaybackDouble("replay_track_peak"), replayAlbumPeak = extras?.optionalPlaybackDouble("replay_album_peak"),
        libraryKind = extras?.getString("library_kind").orEmpty(), chapters = decodePlaybackChapters(extras?.getString("chapters")),
        playbackMode = extras?.getString("playback_mode").orEmpty(), playbackQuality = extras?.getString("playback_quality").orEmpty(),
    )
}

internal class PlaybackSnapshotStore(private val context: Context) {
    private val preferences = context.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)

    fun save(player: Player, durable: Boolean = false) {
        if (playbackSnapshotAction(player.mediaItemCount) == PlaybackSnapshotAction.Clear) {
            clear(durable)
            return
        }
        val current = player.currentMediaItemIndex.coerceAtLeast(0)
        val first = (current - MAX_ITEMS / 2).coerceAtLeast(0).coerceAtMost((player.mediaItemCount - MAX_ITEMS).coerceAtLeast(0))
        val last = minOf(player.mediaItemCount, first + MAX_ITEMS)
        val values = JSONArray()
        for (index in first until last) values.put(encodeItem(player.getMediaItemAt(index), kotlin.math.abs(index - current) <= 1))
        val root = JSONObject().apply {
            put("version", 1); put("items", values); put("start_index", current - first)
            put("position_ms", player.currentPosition.coerceAtLeast(0)); put("speed", player.playbackParameters.speed.toDouble())
            put("repeat_mode", player.repeatMode); put("shuffle", player.shuffleModeEnabled)
        }
        val payload = root.toString()
        if (payload.length > MAX_SNAPSHOT_CHARS) return
        val edit = preferences.edit().putString(KEY_SNAPSHOT, payload)
        if (durable) edit.commit() else edit.apply()
    }

    fun restore(credential: DeviceCredential?): PlaybackSnapshot? = runCatching {
        val root = JSONObject(preferences.getString(KEY_SNAPSHOT, "") ?: "")
        if (root.optInt("version") != 1) return@runCatching null
        val values = root.getJSONArray("items")
        if (values.length() !in 1..MAX_ITEMS) return@runCatching null
        val items = List(values.length()) { index -> decodeItem(values.getJSONObject(index), credential) }
        PlaybackSnapshot(
            items, root.optInt("start_index").coerceIn(items.indices), root.optLong("position_ms").coerceAtLeast(0),
            root.optDouble("speed", 1.0).takeIf { it.isFinite() && it in 0.5..3.0 }?.toFloat() ?: 1f,
            root.optInt("repeat_mode").takeIf { it in Player.REPEAT_MODE_OFF..Player.REPEAT_MODE_ALL } ?: Player.REPEAT_MODE_OFF,
            root.optBoolean("shuffle"),
        )
    }.getOrNull()

    fun clear(durable: Boolean = false) {
        val edit = preferences.edit().remove(KEY_SNAPSHOT)
        if (durable) edit.commit() else edit.apply()
    }

    private fun encodeItem(item: MediaItem, includeChapters: Boolean): JSONObject {
        val track = item.toPlaybackTrack()
        val uri = item.localConfiguration?.uri
        return JSONObject().apply {
            put("id", track.id); put("library_id", track.libraryId); put("library_kind", track.libraryKind)
            put("playback_mode", track.playbackMode); put("playback_quality", track.playbackQuality)
            put("title", track.title.take(500)); put("artist", track.artist.take(500)); put("album", track.album.take(500))
            if (track.durationSeconds.isFinite() && track.durationSeconds > 0.0) put("duration_seconds", track.durationSeconds)
            put("replay_track_db", track.replayTrackDb); put("replay_album_db", track.replayAlbumDb)
            put("replay_track_peak", track.replayTrackPeak); put("replay_album_peak", track.replayAlbumPeak)
            if (includeChapters) put("chapters", JSONArray(encodePlaybackChapters(track.chapters)))
            if (uri?.scheme == "file" || uri?.scheme == "content" || uri?.scheme == WEB_DAV_STREAM_SCHEME) put("local_uri", uri.toString())
        }
    }

    private fun decodeItem(value: JSONObject, credential: DeviceCredential?): MediaItem {
        val track = AudioTrack(
            id = value.getString("id").take(200), libraryId = value.optString("library_id").take(200), title = value.optString("title").take(500),
            artist = value.optString("artist").take(500), album = value.optString("album").take(500),
            durationSeconds = value.optionalFinite("duration_seconds")?.takeIf { it > 0.0 } ?: 0.0,
            replayTrackDb = value.optionalFinite("replay_track_db"), replayAlbumDb = value.optionalFinite("replay_album_db"),
            replayTrackPeak = value.optionalFinite("replay_track_peak"), replayAlbumPeak = value.optionalFinite("replay_album_peak"),
            libraryKind = value.optString("library_kind"), chapters = decodePlaybackChapters(value.optJSONArray("chapters")?.toString()),
            playbackMode = value.optString("playback_mode"), playbackQuality = value.optString("playback_quality"),
        )
        val local = value.optString("local_uri").takeIf {
            val uri = Uri.parse(it)
            when (uri.scheme) {
                "file" -> uri.path?.let(::File)?.isFile == true
                "content" -> runCatching { context.contentResolver.openAssetFileDescriptor(uri, "r")?.use { true } == true }.getOrDefault(false)
                WEB_DAV_STREAM_SCHEME -> parseWebDavStreamToken(uri) != null
                else -> false
            }
        }
        val source = local ?: credential?.let { restoredStreamUrl(it, track) } ?: error("No accessible playback source")
        return track.toPlaybackMediaItem(source)
    }

    companion object {
        private const val PREFERENCES = "playback_snapshot"
        private const val KEY_SNAPSHOT = "active"
        internal const val MAX_ITEMS = 200
        private const val MAX_SNAPSHOT_CHARS = 512 * 1024
    }
}

internal data class AudioPlaybackSource(val mode: String, val quality: String)

internal fun audioPlaybackSource(uri: String): AudioPlaybackSource {
    val path = runCatching { URI(uri).rawPath.orEmpty().split('/').filter(String::isNotBlank) }.getOrDefault(emptyList())
    val marker = path.indexOf("audio-hls")
    val quality = path.getOrNull(marker + 3).orEmpty().takeIf { it in setOf("high", "medium", "low") }.orEmpty()
    return if (marker >= 0 && quality.isNotBlank()) AudioPlaybackSource("converted", quality)
    else AudioPlaybackSource("original", "maximum")
}

private fun restoredStreamUrl(credential: DeviceCredential, track: AudioTrack): String {
    val base = credential.serverUrl.trimEnd('/')
    return if (track.libraryKind == "podcasts") "$base/api/android/v1/podcast-stream/${pathSegment(track.id)}"
    else "$base/api/android/v1/audio-stream/${pathSegment(track.libraryId)}/${pathSegment(track.id)}/audio"
}

private fun pathSegment(value: String) = URLEncoder.encode(value, Charsets.UTF_8.name()).replace("+", "%20")
private fun JSONObject.optionalFinite(key: String) = if (has(key) && !isNull(key)) optDouble(key).takeIf(Double::isFinite) else null
private fun Bundle.optionalPlaybackDouble(key: String) = if (containsKey(key)) getDouble(key).takeIf(Double::isFinite) else null

private fun encodePlaybackChapters(chapters: List<AudioChapter>): String = JSONArray().apply {
    chapters.take(200).forEach { chapter -> put(JSONObject().put("title", chapter.title.take(200)).put("start", chapter.startSeconds).put("end", chapter.endSeconds)) }
}.toString()

private fun decodePlaybackChapters(raw: String?): List<AudioChapter> = runCatching {
    val values = JSONArray(raw ?: "[]")
    List(minOf(values.length(), 200)) { index -> values.getJSONObject(index).run {
        AudioChapter(optString("title", "Chapter ${index + 1}").take(200), optDouble("start"), optDouble("end"))
    } }
}.getOrDefault(emptyList())
